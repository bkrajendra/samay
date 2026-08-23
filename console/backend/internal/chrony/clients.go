package chrony

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// ClientView is one row of `chronyc clients`, shaped for the API/UI.
type ClientView struct {
	Address        string `json:"address"`
	NTPRequests    int    `json:"ntpRequests"`
	LastRequestAgo *int   `json:"lastRequestAgoSecs"` // nil if never
	Status         string `json:"status"`             // Active | Stale | Offline
}

var clientLineRe = regexp.MustCompile(
	`^(\S+)\s+(\d+)\s+(\d+)\s+(-?\d+)\s+(\S+)\s+(\S+)\s+(\d+)\s+(\d+)\s+(\S+)\s+(\S+)\s*$`)

func parseClients(out string) []ClientView {
	var clients []ClientView
	for _, line := range strings.Split(out, "\n") {
		m := clientLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if m[1] == "Hostname" {
			continue // header row, if it happens to match the shape
		}
		ntpRequests, _ := strconv.Atoi(m[2])
		lastNTP, never := parseAgeSecs(m[6])

		var lastPtr *int
		if !never {
			lastPtr = &lastNTP
		}

		clients = append(clients, ClientView{
			Address:        m[1],
			NTPRequests:    ntpRequests,
			LastRequestAgo: lastPtr,
			Status:         clientStatus(lastPtr),
		})
	}
	return clients
}

func clientStatus(lastRequestAgoSecs *int) string {
	if lastRequestAgoSecs == nil {
		return "Offline"
	}
	switch {
	case *lastRequestAgoSecs <= 300: // 5 minutes
		return "Active"
	case *lastRequestAgoSecs <= 86400: // 1 day
		return "Stale"
	default:
		return "Offline"
	}
}

// GetClients runs `chronyc clients` and parses the result.
func (c *Client) GetClients(ctx context.Context) ([]ClientView, error) {
	out, err := c.run(ctx, "clients")
	if err != nil {
		return nil, err
	}
	return parseClients(out), nil
}
