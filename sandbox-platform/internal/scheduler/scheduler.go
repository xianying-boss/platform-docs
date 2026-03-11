// Package scheduler selects which node should run a job.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"github.com/sandbox/platform/pkg/types"
)

// Scheduler routes jobs to the least-loaded available node.
type Scheduler struct {
	rdb      *redis.Client
	selector NodeSelector
}

// New creates a Scheduler backed by Redis for node discovery.
func New(rdb *redis.Client) *Scheduler {
	return &Scheduler{
		rdb:      rdb,
		selector: &LeastLoadedSelector{},
	}
}

// Dispatch enqueues a job onto the selected node's job queue.
// Returns the node ID it was sent to.
func (s *Scheduler) Dispatch(ctx context.Context, job types.Job) (string, error) {
	nodes, err := s.liveNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("list nodes: %w", err)
	}
	if len(nodes) == 0 {
		return "", fmt.Errorf("no available nodes")
	}

	node := s.selector.Select(nodes)
	if node == nil {
		return "", fmt.Errorf("no eligible node for tier %s", job.Tier)
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("marshal job: %w", err)
	}

	queueKey := fmt.Sprintf("node:%s:jobs", node.ID)
	if err := s.rdb.RPush(ctx, queueKey, payload).Err(); err != nil {
		return "", fmt.Errorf("enqueue on node %s: %w", node.ID, err)
	}

	slog.Info("job dispatched", "job_id", job.ID, "node", node.ID, "load", node.Load)
	return node.ID, nil
}

// liveNodes returns all nodes with status="active" from Redis.
func (s *Scheduler) liveNodes(ctx context.Context) ([]types.NodeInfo, error) {
	var cursor uint64
	var nodes []types.NodeInfo
	for {
		keys, next, err := s.rdb.Scan(ctx, cursor, "node:*", 50).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			fields, err := s.rdb.HGetAll(ctx, key).Result()
			if err != nil || fields["status"] != "active" {
				continue
			}
			var load float64
			fmt.Sscanf(fields["load"], "%f", &load)
			nodes = append(nodes, types.NodeInfo{
				ID:      fields["id"],
				Address: fields["address"],
				Load:    load,
			})
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nodes, nil
}
