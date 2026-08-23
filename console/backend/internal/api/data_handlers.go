package api

import (
	"net/http"
	"os"
	"time"
)

type statusResponse struct {
	Running               bool   `json:"running"`
	Synchronized          bool   `json:"synchronized"`
	Stratum               int    `json:"stratum"`
	SourceCount           int    `json:"sourceCount"`
	ReachableSourceCount  int    `json:"reachableSourceCount"`
	ClientCount           int    `json:"clientCount"`
	ServerTimeUTC         string `json:"serverTimeUtc"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := statusResponse{ServerTimeUTC: time.Now().UTC().Format(time.RFC3339)}

	tracking, err := s.Chrony.GetTracking(ctx)
	if err != nil {
		// chronyd unreachable: report it as down rather than failing the
		// request, so the dashboard can still render a "Stopped" state.
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp.Running = true
	resp.Synchronized = tracking.Synchronized
	resp.Stratum = tracking.Stratum

	if sources, err := s.Chrony.GetSources(ctx); err == nil {
		resp.SourceCount = len(sources)
		for _, src := range sources {
			if src.Reach&1 == 1 { // low bit set = most recent poll succeeded
				resp.ReachableSourceCount++
			}
		}
	}

	if clients, err := s.Chrony.GetClients(ctx); err == nil {
		resp.ClientCount = len(clients)
	}

	writeJSON(w, http.StatusOK, resp)
}

type trackingResponse struct {
	RefID              string  `json:"refId"`
	RefName            string  `json:"refName"`
	Stratum            int     `json:"stratum"`
	SystemOffsetSecs   float64 `json:"systemOffsetSecs"`
	LastOffsetSecs     float64 `json:"lastOffsetSecs"`
	RMSOffsetSecs      float64 `json:"rmsOffsetSecs"`
	FrequencyPPM       float64 `json:"frequencyPpm"`
	RootDelaySecs      float64 `json:"rootDelaySecs"`
	RootDispersionSecs float64 `json:"rootDispersionSecs"`
	UpdateIntervalSecs float64 `json:"updateIntervalSecs"`
	LeapStatus         string  `json:"leapStatus"`
	Synchronized       bool    `json:"synchronized"`
	RefTimeUTC         string  `json:"refTimeUtc,omitempty"`
	LastSyncAgoSecs    float64 `json:"lastSyncAgoSecs,omitempty"`
	ServerTimeLocal    string  `json:"serverTimeLocal"`
	ServerTimeUTC      string  `json:"serverTimeUtc"`
	Timezone           string  `json:"timezone"`
}

func (s *Server) handleTracking(w http.ResponseWriter, r *http.Request) {
	tracking, err := s.Chrony.GetTracking(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "chronyd is not responding", err)
		return
	}

	now := time.Now()
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "UTC"
	}

	resp := trackingResponse{
		RefID:              tracking.RefID,
		RefName:            tracking.RefName,
		Stratum:            tracking.Stratum,
		SystemOffsetSecs:   tracking.SystemOffsetSecs,
		LastOffsetSecs:     tracking.LastOffsetSecs,
		RMSOffsetSecs:      tracking.RMSOffsetSecs,
		FrequencyPPM:       tracking.FrequencyPPM,
		RootDelaySecs:      tracking.RootDelaySecs,
		RootDispersionSecs: tracking.RootDispersionSecs,
		UpdateIntervalSecs: tracking.UpdateIntervalSecs,
		LeapStatus:         tracking.LeapStatus,
		Synchronized:       tracking.Synchronized,
		ServerTimeLocal:    now.Format(time.RFC3339),
		ServerTimeUTC:      now.UTC().Format(time.RFC3339),
		Timezone:           tz,
	}
	if !tracking.RefTimeUTC.IsZero() {
		resp.RefTimeUTC = tracking.RefTimeUTC.Format(time.RFC3339)
		resp.LastSyncAgoSecs = now.UTC().Sub(tracking.RefTimeUTC).Seconds()
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.Chrony.GetSources(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "failed to read NTP sources", err)
		return
	}
	writeJSON(w, http.StatusOK, sources)
}

func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	clients, err := s.Chrony.GetClients(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "failed to read NTP clients", err)
		return
	}
	writeJSON(w, http.StatusOK, clients)
}
