package chrony

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// Source is one row of `chronyc sources -v`.
type Source struct {
	Mode               string // "^" server, "=" peer, "#" local clock
	State              string // "*" selected, "+" combined, "-" not combined, "x"/"~"/"?" unusable/variable/new
	Name               string
	Stratum            int
	Poll               int
	Reach              int // decimal, parsed from the octal reach register
	LastRxSecs         int
	LastRxNever        bool
	AdjustedOffsetSecs float64
	MeasuredOffsetSecs float64
	EstErrorSecs       float64
}

// SourceStat is one row of `chronyc sourcestats -v`.
type SourceStat struct {
	Name         string
	SamplePoints int
	ResidualRuns int
	SpanSecs     float64
	FrequencyPPM float64
	FreqSkewPPM  float64
	OffsetSecs   float64
	StdDevSecs   float64
}

// SourceView is a source with its stats merged, shaped for the API/UI.
type SourceView struct {
	Address    string  `json:"address"`
	Stratum    int     `json:"stratum"`
	Reach      int     `json:"reach"`
	LastRxSecs int     `json:"lastRxSecs"`
	OffsetSecs float64 `json:"offsetSecs"`
	JitterSecs float64 `json:"jitterSecs"`
	Status     string  `json:"status"`
}

var durationUnitRe = regexp.MustCompile(`^([+-]?[0-9]*\.?[0-9]+)(ns|us|ms|s)?$`)

func parseDurationSecs(s string) float64 {
	s = strings.TrimSpace(s)
	m := durationUnitRe.FindStringSubmatch(s)
	if m == nil {
		return firstFloat(s)
	}
	v, _ := strconv.ParseFloat(m[1], 64)
	switch m[2] {
	case "ns":
		return v * 1e-9
	case "us":
		return v * 1e-6
	case "ms":
		return v * 1e-3
	default: // "s" or no unit (already seconds)
		return v
	}
}

var ageUnitRe = regexp.MustCompile(`^([0-9]*\.?[0-9]+)(s|m|h|d|y)?$`)

// parseAgeSecs parses chrony's compact age format (e.g. "23", "11m", "3h",
// "10d") into seconds. "-" means never.
func parseAgeSecs(s string) (secs int, never bool) {
	s = strings.TrimSpace(s)
	if s == "-" {
		return 0, true
	}
	m := ageUnitRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	v, _ := strconv.ParseFloat(m[1], 64)
	switch m[2] {
	case "m":
		v *= 60
	case "h":
		v *= 3600
	case "d":
		v *= 86400
	case "y":
		v *= 365 * 86400
	}
	return int(v), false
}

// sourceLineRe captures the two fixed-position mode/state characters
// (which may be a literal space when a source hasn't been evaluated yet),
// then whitespace-delimited fields for the rest of the row. This is more
// robust than fixed-column offsets, which vary slightly by chrony build.
var sourceLineRe = regexp.MustCompile(
	`^(.)(.)\s+(\S+)\s+(-?\d+)\s+(-?\d+)\s+(\d+)\s+(\S+)\s+` +
		`([+-]?[0-9.]+\S*)\[\s*([+-]?[0-9.]+\S*)\]\s*\+/-\s*([0-9.]+\S*)\s*$`)

func parseSourcesVerbose(out string) []Source {
	var sources []Source
	for _, line := range strings.Split(out, "\n") {
		m := sourceLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		stratum, _ := strconv.Atoi(m[4])
		poll, _ := strconv.Atoi(m[5])
		reach, _ := strconv.ParseInt(m[6], 8, 64)
		lastRx, never := parseAgeSecs(m[7])

		sources = append(sources, Source{
			Mode:               m[1],
			State:              strings.TrimSpace(m[2]),
			Name:               m[3],
			Stratum:            stratum,
			Poll:               poll,
			Reach:              int(reach),
			LastRxSecs:         lastRx,
			LastRxNever:        never,
			AdjustedOffsetSecs: parseDurationSecs(m[8]),
			MeasuredOffsetSecs: parseDurationSecs(m[9]),
			EstErrorSecs:       parseDurationSecs(m[10]),
		})
	}
	return sources
}

var sourceStatsLineRe = regexp.MustCompile(
	`^(\S+)\s+(\d+)\s+(\d+)\s+(\S+)\s+([+-]?[0-9.]+)\s+([0-9.]+)\s+([+-]?[0-9.]+\S*)\s+([0-9.]+\S*)\s*$`)

func parseSourceStatsVerbose(out string) []SourceStat {
	var stats []SourceStat
	for _, line := range strings.Split(out, "\n") {
		m := sourceStatsLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		np, _ := strconv.Atoi(m[2])
		nr, _ := strconv.Atoi(m[3])
		span, _ := parseAgeSecs(m[4])
		freq, _ := strconv.ParseFloat(m[5], 64)
		skew, _ := strconv.ParseFloat(m[6], 64)

		stats = append(stats, SourceStat{
			Name:         m[1],
			SamplePoints: np,
			ResidualRuns: nr,
			SpanSecs:     float64(span),
			FrequencyPPM: freq,
			FreqSkewPPM:  skew,
			OffsetSecs:   parseDurationSecs(m[7]),
			StdDevSecs:   parseDurationSecs(m[8]),
		})
	}
	return stats
}

func statusForSource(s Source) string {
	switch s.State {
	case "*":
		return "Selected"
	case "+":
		return "Candidate"
	case "-":
		return "NotCombined"
	case "x":
		return "MayBeInError"
	case "~":
		return "TooVariable"
	case "?":
		return "Unusable"
	default:
		return "Unreachable"
	}
}

// GetSources runs `chronyc sources -v` and `chronyc sourcestats -v` and
// merges them by name into API-ready rows.
func (c *Client) GetSources(ctx context.Context) ([]SourceView, error) {
	sourcesOut, err := c.run(ctx, "sources", "-v")
	if err != nil {
		return nil, err
	}
	statsOut, err := c.run(ctx, "sourcestats", "-v")
	if err != nil {
		return nil, err
	}

	sources := parseSourcesVerbose(sourcesOut)
	stats := parseSourceStatsVerbose(statsOut)
	statsByName := make(map[string]SourceStat, len(stats))
	for _, st := range stats {
		statsByName[st.Name] = st
	}

	views := make([]SourceView, 0, len(sources))
	for _, s := range sources {
		jitter := s.EstErrorSecs
		if st, ok := statsByName[s.Name]; ok {
			jitter = st.StdDevSecs
		}
		views = append(views, SourceView{
			Address:    s.Name,
			Stratum:    s.Stratum,
			Reach:      s.Reach,
			LastRxSecs: s.LastRxSecs,
			OffsetSecs: s.AdjustedOffsetSecs,
			JitterSecs: jitter,
			Status:     statusForSource(s),
		})
	}
	return views, nil
}
