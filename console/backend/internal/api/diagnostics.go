package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type diagnosticCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok | warn | fail
	Message string `json:"message"`
}

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var checks []diagnosticCheck

	tracking, trackErr := s.Chrony.GetTracking(ctx)
	running := trackErr == nil
	checks = append(checks, boolCheck("Chronyd service", running,
		"chronyd is responding on its command socket",
		"chronyd is not responding — it may be stopped or crashed"))

	if running {
		checks = append(checks, boolCheck("System clock synchronized", tracking.Synchronized,
			"system clock is synchronized", "system clock is not synchronized to any source"))

		checks = append(checks, maxThresholdCheck("Stratum", float64(tracking.Stratum), 15,
			fmt.Sprintf("stratum %d", tracking.Stratum),
			"stratum is at or above 15 (unsynchronized)"))

		checks = append(checks, maxThresholdCheck("Root dispersion", tracking.RootDispersionSecs, 0.5,
			"root dispersion is within normal range",
			"root dispersion is high — possible cause: unstable upstream connectivity"))
	}

	if sources, err := s.Chrony.GetSources(ctx); err == nil {
		reachable := 0
		for _, src := range sources {
			if src.Reach&1 == 1 { // low bit set = most recent poll succeeded
				reachable++
			}
		}
		checks = append(checks, boolCheck("Upstream source reachable", reachable > 0,
			fmt.Sprintf("%d of %d sources reachable", reachable, len(sources)),
			"no upstream sources are currently reachable"))
	}

	checks = append(checks, portCheck(123))

	writeJSON(w, http.StatusOK, checks)
}

func boolCheck(name string, ok bool, okMsg, failMsg string) diagnosticCheck {
	if ok {
		return diagnosticCheck{Name: name, Status: "ok", Message: okMsg}
	}
	return diagnosticCheck{Name: name, Status: "fail", Message: failMsg}
}

// maxThresholdCheck warns when value is at or above threshold.
func maxThresholdCheck(name string, value, threshold float64, okMsg, warnMsg string) diagnosticCheck {
	if value >= threshold {
		return diagnosticCheck{Name: name, Status: "warn", Message: warnMsg}
	}
	return diagnosticCheck{Name: name, Status: "ok", Message: okMsg}
}

func portCheck(port int) diagnosticCheck {
	listening := udpPortListening(port)
	return boolCheck(fmt.Sprintf("NTP port %d", port), listening,
		"UDP port is listening", "UDP port does not appear to be listening")
}

// udpPortListening checks /proc/net/udp{,6} for a socket bound to port.
// Containers in the same pod share a network namespace, so this reflects
// chronyd's actual listening socket even though the check runs in the
// console container.
func udpPortListening(port int) bool {
	hexPort := fmt.Sprintf("%04X", port)
	for _, path := range []string{"/proc/net/udp", "/proc/net/udp6"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			// local_address looks like "00000000:007B"
			parts := strings.Split(fields[1], ":")
			if len(parts) == 2 && strings.EqualFold(parts[1], hexPort) {
				return true
			}
		}
	}
	return false
}
