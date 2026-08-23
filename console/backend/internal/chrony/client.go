// Package chrony wraps the chronyc CLI with a fixed set of whitelisted
// operations. It never accepts caller-supplied command strings: every
// function here runs a single, fixed argv against chronyc, so nothing above
// this package can smuggle arbitrary shell/chronyc commands through to the
// host.
package chrony

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Client runs chronyc against a specific command socket.
type Client struct {
	// SocketPath is the path to chronyd's unix command socket, e.g.
	// /run/chrony/chronyd.sock.
	SocketPath string
	// Timeout bounds every chronyc invocation.
	Timeout time.Duration
}

func NewClient(socketPath string) *Client {
	return &Client{SocketPath: socketPath, Timeout: 5 * time.Second}
}

// run executes `chronyc -h <socket> <args...>` and returns stdout. args must
// be a fixed, whitelisted set of tokens chosen by this package's callers,
// never raw user input.
func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	full := append([]string{"-h", c.SocketPath}, args...)
	cmd := exec.CommandContext(ctx, "chronyc", full...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("chronyc %v: %w (stderr: %s)", args, err, stderr.String())
	}
	return stdout.String(), nil
}
