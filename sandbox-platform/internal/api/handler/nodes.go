package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sandbox/platform/pkg/types"
)

// NodesHandler handles GET /v1/nodes.
type NodesHandler struct {
	rdb *redis.Client
}

// NewNodesHandler creates a NodesHandler.
func NewNodesHandler(rdb *redis.Client) *NodesHandler {
	return &NodesHandler{rdb: rdb}
}

// List returns all registered sandbox nodes with their current status.
func (h *NodesHandler) List(c *fiber.Ctx) error {
	// Scan for all node keys: "node:*"
	var cursor uint64
	var keys []string
	for {
		batch, next, err := h.rdb.Scan(c.Context(), cursor, "node:*", 100).Result()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "scan failed")
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}

	var nodes []types.NodeInfo
	for _, key := range keys {
		// Skip sub-keys like node:id:jobs
		if strings.Count(key, ":") > 1 {
			continue
		}
		fields, err := h.rdb.HGetAll(c.Context(), key).Result()
		if err != nil || fields["status"] != "active" {
			continue
		}
		nodes = append(nodes, types.NodeInfo{
			ID:      fields["id"],
			Address: fields["address"],
		})
	}

	return c.JSON(fiber.Map{"nodes": nodes})
}
