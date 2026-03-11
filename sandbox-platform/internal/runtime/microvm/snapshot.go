package microvm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// SnapshotBuilder creates Firecracker VM snapshots for fast resume.
type SnapshotBuilder struct {
	dataDir string
}

// NewSnapshotBuilder creates a SnapshotBuilder that stores snapshots in dataDir.
func NewSnapshotBuilder(dataDir string) (*SnapshotBuilder, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create snapshot dir: %w", err)
	}
	return &SnapshotBuilder{dataDir: dataDir}, nil
}

// Build boots a fresh VM, waits for it to be ready, then takes a snapshot.
// The resulting snapshot is stored at dataDir/name/{mem,state}.
func (sb *SnapshotBuilder) Build(ctx context.Context, name, kernelPath, rootfsPath string) (string, error) {
	snapshotDir := filepath.Join(sb.dataDir, name)
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	vm, err := NewVM(ctx, "snapshot-builder-"+name, rootfsPath, kernelPath)
	if err != nil {
		return "", fmt.Errorf("boot vm: %w", err)
	}
	defer vm.Stop()

	// Pause the VM before snapshot.
	if err := vm.pause(ctx); err != nil {
		return "", fmt.Errorf("pause vm: %w", err)
	}

	// Issue snapshot request via Firecracker API.
	snapReq := map[string]any{
		"snapshot_type":    "Full",
		"snapshot_path":    filepath.Join(snapshotDir, "state"),
		"mem_file_path":    filepath.Join(snapshotDir, "mem"),
		"version":          "0.23.0",
	}
	if err := vm.apiPut(ctx, "/snapshot/create", snapReq); err != nil {
		return "", fmt.Errorf("snapshot: %w", err)
	}

	return snapshotDir, nil
}

// pause sends the Firecracker VM pause request.
func (v *VM) pause(ctx context.Context) error {
	return v.apiPut(ctx, "/vm", map[string]string{"state": "Paused"})
}

// apiPut sends a PUT request to the Firecracker unix socket API.
func (v *VM) apiPut(ctx context.Context, path string, body any) error {
	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		"http://localhost"+path, jsonReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("firecracker API %s → %d: %s", path, resp.StatusCode, b)
	}
	return nil
}
