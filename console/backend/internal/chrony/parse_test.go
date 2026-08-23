package chrony

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-12
}

func TestParseTracking(t *testing.T) {
	out := `Reference ID    : C0A80105 (192.168.1.5)
Stratum         : 3
Ref time (UTC)  : Thu Jan 01 00:00:00 2026
System time     : 0.000012345 seconds fast of NTP time
Last offset     : +0.000006789 seconds
RMS offset      : 0.000045678 seconds
Frequency       : 3.456 ppm slow
Residual freq   : +0.001 ppm
Skew            : 0.123 ppm
Root delay      : 0.001234567 seconds
Root dispersion : 0.000987654 seconds
Update interval : 64.5 seconds
Leap status     : Normal
`
	tr := parseTracking(out)

	if tr.RefID != "C0A80105" || tr.RefName != "192.168.1.5" {
		t.Errorf("RefID/RefName = %q/%q", tr.RefID, tr.RefName)
	}
	if tr.Stratum != 3 {
		t.Errorf("Stratum = %d, want 3", tr.Stratum)
	}
	if !almostEqual(tr.SystemOffsetSecs, 0.000012345) {
		t.Errorf("SystemOffsetSecs = %v, want +0.000012345 (fast = positive)", tr.SystemOffsetSecs)
	}
	if !almostEqual(tr.FrequencyPPM, -3.456) {
		t.Errorf("FrequencyPPM = %v, want -3.456 (slow = negative)", tr.FrequencyPPM)
	}
	if tr.LeapStatus != "Normal" || !tr.Synchronized {
		t.Errorf("LeapStatus/Synchronized = %q/%v", tr.LeapStatus, tr.Synchronized)
	}
	if tr.RefTimeUTC.Year() != 2026 {
		t.Errorf("RefTimeUTC = %v, want year 2026", tr.RefTimeUTC)
	}
}

func TestParseTrackingNotSynchronised(t *testing.T) {
	out := "Leap status     : Not synchronised\n"
	tr := parseTracking(out)
	if tr.Synchronized {
		t.Error("Synchronized = true, want false for 'Not synchronised'")
	}
}

func TestParseSourcesVerbose(t *testing.T) {
	out := `MS Name/IP address         Stratum Poll Reach LastRx Last sample
===============================================================================
^* GPS0                          0   4   377    11  -6ns[  -6ns] +/-  219ns
^+ wolf.blueyonder.co.uk         2   7   377    13   -161us[ -172us] +/- 4629us
^- foo.example.net                1   6   377    47   +100us[+9155us] +/-   19ms
`
	sources := parseSourcesVerbose(out)
	if len(sources) != 3 {
		t.Fatalf("got %d sources, want 3", len(sources))
	}

	gps := sources[0]
	if gps.Name != "GPS0" || gps.Mode != "^" || gps.State != "*" {
		t.Errorf("gps = %+v", gps)
	}
	if gps.Stratum != 0 || gps.Poll != 4 || gps.Reach != 0377 || gps.LastRxSecs != 11 {
		t.Errorf("gps fields = %+v", gps)
	}
	if !almostEqual(gps.AdjustedOffsetSecs, -6e-9) {
		t.Errorf("gps.AdjustedOffsetSecs = %v, want -6ns", gps.AdjustedOffsetSecs)
	}
	if !almostEqual(gps.EstErrorSecs, 219e-9) {
		t.Errorf("gps.EstErrorSecs = %v, want 219ns", gps.EstErrorSecs)
	}

	wolf := sources[1]
	if wolf.State != "+" || statusForSource(wolf) != "Candidate" {
		t.Errorf("wolf.State = %q", wolf.State)
	}

	foo := sources[2]
	if foo.State != "-" || !almostEqual(foo.MeasuredOffsetSecs, 9155e-6) {
		t.Errorf("foo = %+v", foo)
	}
}

func TestParseSourceStatsVerbose(t *testing.T) {
	out := `Name/IP Address            NP  NR  Span  Frequency  Freq Skew  Offset  Std Dev
==============================================================================
GPS0                         8   5    62      0.001      0.045     -2345ns   365ns
`
	stats := parseSourceStatsVerbose(out)
	if len(stats) != 1 {
		t.Fatalf("got %d stats, want 1", len(stats))
	}
	s := stats[0]
	if s.Name != "GPS0" || s.SamplePoints != 8 || s.ResidualRuns != 5 {
		t.Errorf("s = %+v", s)
	}
	if !almostEqual(s.OffsetSecs, -2345e-9) {
		t.Errorf("s.OffsetSecs = %v, want -2345ns", s.OffsetSecs)
	}
	if !almostEqual(s.StdDevSecs, 365e-9) {
		t.Errorf("s.StdDevSecs = %v, want 365ns", s.StdDevSecs)
	}
}

func TestParseClients(t *testing.T) {
	out := `Hostname                      NTP   Drop Int IntL Last     Cmd   Drop Int  Last
===============================================================================
192.168.10.21                 1823      0   6    -     12      0     0    -    -
192.168.10.42                    91      0  10    -  86400      0     0    -    -
203.0.113.5                       4      0   6    -      -      0     0    -    -
`
	clients := parseClients(out)
	if len(clients) != 3 {
		t.Fatalf("got %d clients, want 3", len(clients))
	}

	active := clients[0]
	if active.Address != "192.168.10.21" || active.NTPRequests != 1823 {
		t.Errorf("active = %+v", active)
	}
	if active.LastRequestAgo == nil || *active.LastRequestAgo != 12 || active.Status != "Active" {
		t.Errorf("active status = %+v", active)
	}

	stale := clients[1]
	if stale.LastRequestAgo == nil || *stale.LastRequestAgo != 86400 || stale.Status != "Stale" {
		t.Errorf("stale status = %+v", stale)
	}

	never := clients[2]
	if never.LastRequestAgo != nil || never.Status != "Offline" {
		t.Errorf("never status = %+v", never)
	}
}

func TestParseAgeSecs(t *testing.T) {
	cases := []struct {
		in    string
		want  int
		never bool
	}{
		{"23", 23, false},
		{"-", 0, true},
		{"11m", 660, false},
		{"3h", 10800, false},
		{"2d", 172800, false},
	}
	for _, c := range cases {
		got, never := parseAgeSecs(c.in)
		if got != c.want || never != c.never {
			t.Errorf("parseAgeSecs(%q) = (%d, %v), want (%d, %v)", c.in, got, never, c.want, c.never)
		}
	}
}
