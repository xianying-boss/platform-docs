package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sandbox/platform/pkg/types"
)

// jobResultKey is the Redis key pattern where results are published.
const jobResultKey = "job:result:%s"

// Executor handles a single job dispatch lifecycle.
type jobExecutor struct {
	nodeID  string
	rdb     *redis.Client
	runtime *RuntimeManager
}

// NewExecutor creates a job executor.
func NewExecutor(nodeID string, rdb *redis.Client, runtime *RuntimeManager) (*jobExecutor, error) {
	return &jobExecutor{nodeID: nodeID, rdb: rdb, runtime: runtime}, nil
}

// Execute decodes the raw JSON payload, routes it to the right runtime,
// and publishes the result back to Redis.
func (e *jobExecutor) Execute(ctx context.Context, payload string) error {
	var job types.Job
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return fmt.Errorf("decode job: %w", err)
	}

	log := slog.With("job_id", job.ID, "tool", job.Tool, "tier", job.Tier)
	log.Info("executing job")

	start := time.Now()

	rt, err := e.runtime.RuntimeFor(job.Tier)
	if err != nil {
		return e.fail(ctx, job, fmt.Errorf("route: %w", err))
	}

	result, err := rt.Execute(job)
	if err != nil {
		return e.fail(ctx, job, fmt.Errorf("runtime: %w", err))
	}

	result.OutputKey = fmt.Sprintf("jobs/%s/output/result.json", job.ID)
	job.Status = types.StatusCompleted
	job.DurationMs = time.Since(start).Milliseconds()
	job.Logs = result.Stdout + result.Stderr

	return e.publishResult(ctx, job, result)
}

// fail marks a job as failed and publishes the error result.
func (e *jobExecutor) fail(ctx context.Context, job types.Job, err error) error {
	slog.Error("job failed", "job_id", job.ID, "err", err)
	job.Status = types.StatusFailed
	job.ErrorMsg = err.Error()
	return e.publishResult(ctx, job, types.RuntimeResult{})
}

// publishResult serialises the completed job and pushes it to Redis.
func (e *jobExecutor) publishResult(ctx context.Context, job types.Job, result types.RuntimeResult) error {
	type envelope struct {
		Job    types.Job            `json:"job"`
		Result types.RuntimeResult  `json:"result"`
	}
	data, err := json.Marshal(envelope{Job: job, Result: result})
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	key := fmt.Sprintf(jobResultKey, job.ID)
	return e.rdb.Set(ctx, key, data, 24*time.Hour).Err()
}
