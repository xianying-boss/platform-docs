package types

import "time"

// Tier represents the execution runtime tier.
type Tier string

const (
	TierWASM    Tier = "wasm"
	TierMicroVM Tier = "microvm"
	TierGUI     Tier = "gui"
)

// JobStatus represents the lifecycle state of a job.
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

// Job is the core execution unit passed between components.
type Job struct {
	ID          string          `json:"id"`
	Tool        string          `json:"tool"`
	Tier        Tier            `json:"tier"`
	Input       map[string]any  `json:"input"`
	Status      JobStatus       `json:"status"`
	OutputRef   string          `json:"output_ref,omitempty"`
	ErrorMsg    string          `json:"error_message,omitempty"`
	Logs        string          `json:"logs,omitempty"`
	DurationMs  int64           `json:"duration_ms,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ExecuteRequest is the API payload for POST /execute.
type ExecuteRequest struct {
	Tool  string         `json:"tool"`
	Input map[string]any `json:"input"`
}

// ExecuteResponse is the API response for a submitted job.
type ExecuteResponse struct {
	JobID string `json:"job_id"`
}

// ToolManifest describes a tool's runtime requirements.
type ToolManifest struct {
	Name        string            `json:"name"`
	Tier        Tier              `json:"tier"`
	Entrypoint  string            `json:"entrypoint"`
	TimeoutSecs int               `json:"timeout"`
	Env         map[string]string `json:"env,omitempty"`
}

// NodeInfo holds live state of a sandbox node.
type NodeInfo struct {
	ID       string  `json:"id"`
	Address  string  `json:"address"`
	Load     float64 `json:"load"`     // 0.0–1.0
	Capacity int     `json:"capacity"` // max concurrent jobs
	Active   int     `json:"active"`   // current active jobs
}

// RuntimeResult is returned by any runtime executor.
type RuntimeResult struct {
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
	OutputKey string `json:"output_key,omitempty"` // MinIO object key
}
