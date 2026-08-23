package chrony

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// ForceSync requests a burst of extra measurements from every source so
// chronyd converges on an accurate offset faster than waiting for its
// normal polling interval.
func (c *Client) ForceSync(ctx context.Context) error {
	_, err := c.run(ctx, "burst", "4/4")
	return err
}

// StepClock immediately steps the system clock to chronyd's current best
// estimate, rather than slewing it gradually. This can cause a visible
// clock jump; callers must confirm with the operator first.
func (c *Client) StepClock(ctx context.Context) error {
	_, err := c.run(ctx, "makestep")
	return err
}

// RestartService sends SIGTERM to the chronyd process. This relies on the
// pod having shareProcessNamespace: true so the console container can see
// chronyd's PID; because chronyd is its own container's PID 1, Kubernetes'
// normal container restart policy brings it back up. No Kubernetes API
// access is required.
func (c *Client) RestartService(ctx context.Context) error {
	pid, err := findProcessByComm("chronyd")
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find chronyd process %d: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal chronyd process %d: %w", pid, err)
	}
	return nil
}

// findProcessByComm scans /proc for a process whose comm matches name
// exactly. Requires shareProcessNamespace: true on the pod.
func findProcessByComm(name string) (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, fmt.Errorf("read /proc: %w", err)
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // not a PID directory
		}
		commBytes, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil {
			continue // process may have exited; skip
		}
		if strings.TrimSpace(string(commBytes)) == name {
			return pid, nil
		}
	}
	return 0, fmt.Errorf("no process named %q found", name)
}
