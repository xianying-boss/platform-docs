// Package gui runs tools in Docker containers with a virtual display (Xvfb).
// The desktop-runner image must be pre-built and available on the node.
package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sandbox/platform/pkg/types"
)

const desktopImage = "sandbox/desktop-runner:latest"

// DockerRunner manages the lifecycle of a single GUI container.
type DockerRunner struct {
	containerID string
	display     string
}

// Start launches a new desktop-runner container.
func (d *DockerRunner) Start(ctx context.Context) error {
	id := "gui-" + uuid.New().String()[:8]
	d.display = ":99"

	//nolint:gosec
	out, err := exec.CommandContext(ctx, "docker", "run",
		"-d",
		"--name", id,
		"--rm",
		"-e", "DISPLAY="+d.display,
		"-e", "XVFB_WHD=1920x1080x24",
		"--shm-size", "256m",
		desktopImage,
	).Output()
	if err != nil {
		return fmt.Errorf("docker run: %w", err)
	}

	d.containerID = strings.TrimSpace(string(out))
	slog.Info("gui container started", "id", d.containerID)
	return nil
}

// Exec runs a command inside the GUI container.
func (d *DockerRunner) Exec(ctx context.Context, cmd string, args []string, env map[string]string) (string, error) {
	dockerArgs := []string{"exec"}
	for k, v := range env {
		dockerArgs = append(dockerArgs, "-e", k+"="+v)
	}
	dockerArgs = append(dockerArgs, d.containerID, cmd)
	dockerArgs = append(dockerArgs, args...)

	//nolint:gosec
	out, err := exec.CommandContext(ctx, "docker", dockerArgs...).CombinedOutput()
	return string(out), err
}

// Stop removes the container.
func (d *DockerRunner) Stop(ctx context.Context) error {
	if d.containerID == "" {
		return nil
	}
	return exec.CommandContext(ctx, "docker", "rm", "-f", d.containerID).Run()
}

// Config holds GUI runtime configuration.
type Config struct {
	PoolSize int
}

// Runtime executes GUI-tier tools in Docker containers.
type Runtime struct {
	cfg  Config
	pool *ContainerPool
}

// NewRuntime creates a GUI runtime with a warm container pool.
func NewRuntime(cfg Config) (*Runtime, error) {
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 2
	}
	pool := NewContainerPool(cfg.PoolSize)
	return &Runtime{cfg: cfg, pool: pool}, nil
}

// Execute runs a GUI tool inside a Docker container.
func (r *Runtime) Execute(job types.Job) (types.RuntimeResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	runner, err := r.pool.Acquire(ctx)
	if err != nil {
		return types.RuntimeResult{}, fmt.Errorf("acquire container: %w", err)
	}
	defer r.pool.Release(ctx, runner)

	inputJSON, _ := json.Marshal(job.Input)
	output, err := runner.Exec(ctx, "/tool/"+job.Tool, nil, map[string]string{
		"TOOL_INPUT": string(inputJSON),
		"DISPLAY":    runner.display,
	})
	exitCode := 0
	if err != nil {
		exitCode = 1
	}
	return types.RuntimeResult{Stdout: output, ExitCode: exitCode}, err
}
