package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sandbox/platform/internal/api/middleware"
	"github.com/sandbox/platform/internal/queue"
	"github.com/sandbox/platform/internal/tool/registry"
	"github.com/sandbox/platform/pkg/types"
)

// ExecuteHandler handles POST /v1/execute.
type ExecuteHandler struct {
	producer *queue.Producer
	tools    *registry.Registry
}

// NewExecuteHandler creates an ExecuteHandler.
func NewExecuteHandler(producer *queue.Producer, tools *registry.Registry) *ExecuteHandler {
	return &ExecuteHandler{producer: producer, tools: tools}
}

// Handle validates the request, enqueues the job, and returns 202 + job_id.
func (h *ExecuteHandler) Handle(c *fiber.Ctx) error {
	var req types.ExecuteRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Tool == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tool is required")
	}

	// Look up tool manifest to determine tier.
	manifest, err := h.tools.Get(req.Tool)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "unknown tool: "+req.Tool)
	}

	// Inject tier into locals so the rate-limit middleware can read it.
	// This is the correct fix for original Bug 1.
	c.Locals("tier", string(manifest.Tier))

	agentID, _ := c.Locals(middleware.ContextKeyAgentID).(string)
	jobID := uuid.New().String()

	inputBytes, _ := json.Marshal(req.Input)

	job := types.Job{
		ID:     jobID,
		Tool:   req.Tool,
		Tier:   manifest.Tier,
		Status: types.StatusPending,
	}
	if err := json.Unmarshal(inputBytes, &job.Input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid input")
	}

	msg := queue.JobMessage{
		JobID:   jobID,
		Tool:    req.Tool,
		Tier:    string(manifest.Tier),
		AgentID: agentID,
		Input:   string(inputBytes),
	}

	if err := h.producer.Push(c.Context(), msg); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to enqueue job")
	}

	return c.Status(fiber.StatusAccepted).JSON(types.ExecuteResponse{JobID: jobID})
}
