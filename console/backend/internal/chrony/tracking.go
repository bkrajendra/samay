package chrony

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Tracking mirrors the fields reported by `chronyc tracking`.
type Tracking struct {
	RefID              string
	RefName            string
	Stratum            int
	RefTimeUTC         time.Time
	SystemOffsetSecs   float64 // positive = system clock ahead of NTP time
	LastOffsetSecs     float64
	RMSOffsetSecs      float64
	FrequencyPPM       float64 // positive = clock runs fast
	ResidualFreqPPM    float64
	SkewPPM            float64
	RootDelaySecs      float64
	RootDispersionSecs float64
	UpdateIntervalSecs float64
	LeapStatus         string
	Synchronized       bool
}

var floatRe = regexp.MustCompile(`[-+]?[0-9]*\.?[0-9]+`)

func firstFloat(s string) float64 {
	m := floatRe.FindString(s)
	if m == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(m, 64)
	return v
}

// signedByDirection parses a magnitude followed by a "slow"/"fast" word and
// returns it signed such that positive always means "ahead"/"fast".
func signedByDirection(s string) float64 {
	v := firstFloat(s)
	if v < 0 {
		v = -v
	}
	if strings.Contains(s, "slow") {
		return -v
	}
	return v
}

func parseRefID(s string) (id, name string) {
	s = strings.TrimSpace(s)
	open := strings.Index(s, "(")
	shut := strings.LastIndex(s, ")")
	if open >= 0 && shut > open {
		return strings.TrimSpace(s[:open]), strings.TrimSpace(s[open+1 : shut])
	}
	return s, ""
}

// GetTracking runs `chronyc tracking` and parses the result.
func (c *Client) GetTracking(ctx context.Context) (Tracking, error) {
	out, err := c.run(ctx, "tracking")
	if err != nil {
		return Tracking{}, err
	}
	return parseTracking(out), nil
}

func parseTracking(out string) Tracking {
	t := Tracking{}
	for _, line := range strings.Split(out, "\n") {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		label := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		switch label {
		case "Reference ID":
			t.RefID, t.RefName = parseRefID(value)
		case "Stratum":
			t.Stratum, _ = strconv.Atoi(value)
		case "Ref time (UTC)":
			// e.g. "Thu Jan 01 00:00:00 2026"
			if parsed, err := time.Parse("Mon Jan 02 15:04:05 2006", value); err == nil {
				t.RefTimeUTC = parsed
			}
		case "System time":
			t.SystemOffsetSecs = signedByDirection(value)
		case "Last offset":
			t.LastOffsetSecs = firstFloat(value)
		case "RMS offset":
			t.RMSOffsetSecs = firstFloat(value)
		case "Frequency":
			t.FrequencyPPM = signedByDirection(value)
		case "Residual freq":
			t.ResidualFreqPPM = firstFloat(value)
		case "Skew":
			t.SkewPPM = firstFloat(value)
		case "Root delay":
			t.RootDelaySecs = firstFloat(value)
		case "Root dispersion":
			t.RootDispersionSecs = firstFloat(value)
		case "Update interval":
			t.UpdateIntervalSecs = firstFloat(value)
		case "Leap status":
			t.LeapStatus = value
		}
	}
	t.Synchronized = t.LeapStatus != "" && t.LeapStatus != "Not synchronised"
	return t
}
