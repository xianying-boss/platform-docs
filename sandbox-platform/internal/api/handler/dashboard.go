package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sandbox/platform/internal/tool/registry"
	"github.com/sandbox/platform/pkg/types"
)

// DashboardHandler handles GET /v1/dashboard — returns full platform state.
type DashboardHandler struct {
	rdb   *redis.Client
	tools *registry.Registry
}

// NewDashboardHandler creates a DashboardHandler.
func NewDashboardHandler(rdb *redis.Client, tools *registry.Registry) *DashboardHandler {
	return &DashboardHandler{rdb: rdb, tools: tools}
}

// ─── Response shapes ─────────────────────────────────────────────────────────

// DashboardResponse is the single payload for GET /v1/dashboard.
type DashboardResponse struct {
	Timestamp   time.Time       `json:"timestamp"`
	Cluster     ClusterSummary  `json:"cluster"`
	Nodes       []NodeDetail    `json:"nodes"`
	Pools       PoolSummary     `json:"pools"`
	Queue       QueueSummary    `json:"queue"`
	Jobs        JobsSummary     `json:"jobs"`
	Tools       []ToolStatus    `json:"tools"`
}

type ClusterSummary struct {
	TotalNodes   int     `json:"total_nodes"`
	ActiveNodes  int     `json:"active_nodes"`
	OfflineNodes int     `json:"offline_nodes"`
	AvgLoad      float64 `json:"avg_load"`
}

type NodeDetail struct {
	ID          string    `json:"id"`
	Address     string    `json:"address"`
	Status      string    `json:"status"`     // active | offline
	Load        float64   `json:"load"`        // 0.0–1.0
	LastSeen    string    `json:"last_seen"`
	RegisteredAt string   `json:"registered_at"`
	QueueDepth  int64     `json:"queue_depth"` // jobs waiting for this node
	Runtimes    RuntimeCapacity `json:"runtimes"`
}

type RuntimeCapacity struct {
	WASM    RuntimeStat `json:"wasm"`
	MicroVM RuntimeStat `json:"microvm"`
	GUI     RuntimeStat `json:"gui"`
}

type RuntimeStat struct {
	Active   int `json:"active"`
	Capacity int `json:"capacity"`
	PoolSize int `json:"pool_size"`
}

type PoolSummary struct {
	WASM    PoolDetail `json:"wasm"`
	MicroVM PoolDetail `json:"microvm"`
	GUI     PoolDetail `json:"gui"`
}

type PoolDetail struct {
	Tier      string  `json:"tier"`
	PoolSize  int     `json:"pool_size"`
	Active    int     `json:"active"`
	Available int     `json:"available"`
	Utilization float64 `json:"utilization"` // 0.0–1.0
}

type QueueSummary struct {
	GlobalDepth  int64            `json:"global_depth"`
	PerNode      map[string]int64 `json:"per_node"`
}

type JobsSummary struct {
	TotalCompleted int64 `json:"total_completed"`
	TotalFailed    int64 `json:"total_failed"`
	TotalPending   int64 `json:"total_pending"`
}

type ToolStatus struct {
	Name     string     `json:"name"`
	Tier     types.Tier `json:"tier"`
	Timeout  int        `json:"timeout"`
	Entrypoint string   `json:"entrypoint"`
}

// ─── Handler ──────────────────────────────────────────────────────────────────

// Get returns the full platform dashboard payload.
func (h *DashboardHandler) Get(c *fiber.Ctx) error {
	ctx := c.Context()

	nodes, err := h.scanNodes(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "scan nodes: "+err.Error())
	}

	queue, err := h.queueSummary(ctx, nodes)
	if err != nil {
		queue = QueueSummary{PerNode: map[string]int64{}}
	}

	jobs, _ := h.jobsSummary(ctx)
	pools   := h.poolSummary(nodes)
	cluster := h.clusterSummary(nodes)

	manifests := h.tools.All()
	toolStatuses := make([]ToolStatus, 0, len(manifests))
	for _, m := range manifests {
		toolStatuses = append(toolStatuses, ToolStatus{
			Name:       m.Name,
			Tier:       m.Tier,
			Timeout:    m.TimeoutSecs,
			Entrypoint: m.Entrypoint,
		})
	}

	return c.JSON(DashboardResponse{
		Timestamp: time.Now().UTC(),
		Cluster:   cluster,
		Nodes:     nodes,
		Pools:     pools,
		Queue:     queue,
		Jobs:      jobs,
		Tools:     toolStatuses,
	})
}

// ─── Private helpers ──────────────────────────────────────────────────────────

func (h *DashboardHandler) scanNodes(ctx context.Context) ([]NodeDetail, error) {
	var cursor uint64
	var details []NodeDetail

	for {
		keys, next, err := h.rdb.Scan(ctx, cursor, "node:*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			// Skip sub-keys like node:id:jobs
			if strings.Count(key, ":") > 1 {
				continue
			}
			fields, err := h.rdb.HGetAll(ctx, key).Result()
			if err != nil || len(fields) == 0 {
				continue
			}

			load, _ := strconv.ParseFloat(fields["load"], 64)

			// Queue depth for this node
			queueKey := fmt.Sprintf("node:%s:jobs", fields["id"])
			queueDepth, _ := h.rdb.LLen(ctx, queueKey).Result()

			details = append(details, NodeDetail{
				ID:           fields["id"],
				Address:      fields["address"],
				Status:       fields["status"],
				Load:         load,
				LastSeen:     fields["last_seen"],
				RegisteredAt: fields["registered_at"],
				QueueDepth:   queueDepth,
				Runtimes: RuntimeCapacity{
					WASM:    RuntimeStat{Capacity: 10000, PoolSize: 0},
					MicroVM: RuntimeStat{Capacity: 100, PoolSize: 10},
					GUI:     RuntimeStat{Capacity: 20, PoolSize: 3},
				},
			})
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return details, nil
}

func (h *DashboardHandler) clusterSummary(nodes []NodeDetail) ClusterSummary {
	active, offline := 0, 0
	totalLoad := 0.0
	for _, n := range nodes {
		if n.Status == "active" {
			active++
			totalLoad += n.Load
		} else {
			offline++
		}
	}
	avg := 0.0
	if active > 0 {
		avg = totalLoad / float64(active)
	}
	return ClusterSummary{
		TotalNodes:   len(nodes),
		ActiveNodes:  active,
		OfflineNodes: offline,
		AvgLoad:      avg,
	}
}

func (h *DashboardHandler) poolSummary(nodes []NodeDetail) PoolSummary {
	var wasmCap, vmCap, guiCap int
	var wasmActive, vmActive, guiActive int
	for _, n := range nodes {
		if n.Status != "active" {
			continue
		}
		wasmCap   += n.Runtimes.WASM.Capacity
		vmCap     += n.Runtimes.MicroVM.Capacity
		guiCap    += n.Runtimes.GUI.Capacity
		wasmActive  += n.Runtimes.WASM.Active
		vmActive    += n.Runtimes.MicroVM.Active
		guiActive   += n.Runtimes.GUI.Active
	}
	util := func(active, cap int) float64 {
		if cap == 0 {
			return 0
		}
		return float64(active) / float64(cap)
	}
	return PoolSummary{
		WASM:    PoolDetail{Tier: "wasm",    PoolSize: 0,  Active: wasmActive,  Available: wasmCap - wasmActive,  Utilization: util(wasmActive, wasmCap),  },
		MicroVM: PoolDetail{Tier: "microvm", PoolSize: 10, Active: vmActive,    Available: vmCap - vmActive,      Utilization: util(vmActive, vmCap),      },
		GUI:     PoolDetail{Tier: "gui",     PoolSize: 3,  Active: guiActive,   Available: guiCap - guiActive,    Utilization: util(guiActive, guiCap),    },
	}
}

func (h *DashboardHandler) queueSummary(ctx context.Context, nodes []NodeDetail) (QueueSummary, error) {
	perNode := make(map[string]int64)
	var total int64
	for _, n := range nodes {
		key := fmt.Sprintf("node:%s:jobs", n.ID)
		depth, err := h.rdb.LLen(ctx, key).Result()
		if err != nil {
			continue
		}
		perNode[n.ID] = depth
		total += depth
	}

	// Also count the global queue
	globalDepth, _ := h.rdb.LLen(ctx, "jobs").Result()
	total += globalDepth

	return QueueSummary{GlobalDepth: total, PerNode: perNode}, nil
}

func (h *DashboardHandler) jobsSummary(ctx context.Context) (JobsSummary, error) {
	// Count job result keys in Redis as a proxy for completed/failed jobs.
	var cursor uint64
	var completed, failed int64
	for {
		keys, next, err := h.rdb.Scan(ctx, cursor, "job:result:*", 200).Result()
		if err != nil {
			return JobsSummary{}, err
		}
		for _, key := range keys {
			raw, err := h.rdb.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}
			var envelope struct {
				Job struct {
					Status string `json:"status"`
				} `json:"job"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				continue
			}
			switch envelope.Job.Status {
			case "completed":
				completed++
			case "failed":
				failed++
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	pending, _ := h.rdb.LLen(ctx, "jobs").Result()
	return JobsSummary{TotalCompleted: completed, TotalFailed: failed, TotalPending: pending}, nil
}
