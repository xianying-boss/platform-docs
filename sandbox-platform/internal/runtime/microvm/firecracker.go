// Package microvm manages Firecracker microVM lifecycle.
// Firecracker binary must be installed at /usr/bin/firecracker.
// Target cold start: <80ms (snapshot resume).
package microvm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	firecrackerBin    = "/usr/bin/firecracker"
	firecrackerSocket = "/tmp/firecracker-%s.sock"
)

// VM represents a running Firecracker microVM.
type VM struct {
	id         string
	socketPath string
	proc       *exec.Cmd
	client     *http.Client
}

// NewVM boots a Firecracker VM from a snapshot. vmID must be unique.
func NewVM(ctx context.Context, vmID, snapshotPath, kernelPath string) (*VM, error) {
	socketPath := fmt.Sprintf(firecrackerSocket, vmID)

	//nolint:gosec
	cmd := exec.CommandContext(ctx, firecrackerBin,
		"--api-sock", socketPath,
		"--config-file", buildVMConfig(vmID, snapshotPath, kernelPath),
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start firecracker: %w", err)
	}

	// Unix socket HTTP client.
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	vm := &VM{id: vmID, socketPath: socketPath, proc: cmd, client: client}

	// Wait until the VM's API is ready (up to 2s).
	if err := vm.waitReady(ctx, 2*time.Second); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("vm not ready: %w", err)
	}

	return vm, nil
}

// Exec sends a command to run inside the VM via the guest agent.
func (v *VM) Exec(ctx context.Context, cmd string, args []string, env map[string]string) (string, int, error) {
	payload, _ := json.Marshal(map[string]any{
		"cmd":  cmd,
		"args": args,
		"env":  env,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		"http://localhost/exec", jsonReader(payload))
	if err != nil {
		return "", -1, fmt.Errorf("build exec request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return "", -1, fmt.Errorf("exec rpc: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", -1, fmt.Errorf("decode exec response: %w", err)
	}
	return result.Output, result.ExitCode, nil
}

// Stop terminates the VM process.
func (v *VM) Stop() error {
	if v.proc != nil && v.proc.Process != nil {
		if err := v.proc.Process.Kill(); err != nil {
			return fmt.Errorf("kill vm process: %w", err)
		}
	}
	if err := os.Remove(v.socketPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("remove socket", "path", v.socketPath, "err", err)
	}
	return nil
}

// waitReady polls the Firecracker API until it responds or timeout expires.
func (v *VM) waitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/", nil)
		resp, err := v.client.Do(req)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for firecracker API")
}

// buildVMConfig writes a minimal Firecracker JSON config to a temp file.
func buildVMConfig(vmID, snapshotPath, kernelPath string) string {
	cfg := map[string]any{
		"boot-source": map[string]any{
			"kernel_image_path": kernelPath,
			"boot_args":         "console=ttyS0 reboot=k panic=1 pci=off",
		},
		"drives": []map[string]any{
			{
				"drive_id":       "rootfs",
				"path_on_host":   snapshotPath,
				"is_root_device": true,
				"is_read_only":   false,
			},
		},
		"machine-config": map[string]any{
			"vcpu_count":  2,
			"mem_size_mib": 512,
		},
	}
	data, _ := json.Marshal(cfg)
	path := filepath.Join(os.TempDir(), "fc-config-"+vmID+".json")
	_ = os.WriteFile(path, data, 0o600)
	return path
}
