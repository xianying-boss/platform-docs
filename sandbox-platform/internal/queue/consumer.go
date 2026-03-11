package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// HandlerFunc processes a single job message.
type HandlerFunc func(ctx context.Context, msg JobMessage) error

// Consumer reads job messages from a Redis list and calls the handler.
type Consumer struct {
	rdb     *redis.Client
	stream  string
	handler HandlerFunc
}

// NewConsumer creates a Consumer.
func NewConsumer(rdb *redis.Client, stream string, handler HandlerFunc) *Consumer {
	return &Consumer{rdb: rdb, stream: stream, handler: handler}
}

// Run blocks, processing messages until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	slog.Info("queue consumer started", "stream", c.stream)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// BLPOP with 1-second timeout so we can check ctx regularly.
		result, err := c.rdb.BLPop(ctx, time.Second, c.stream).Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				continue
			}
			slog.Error("blpop", "err", err)
			continue
		}
		if len(result) < 2 {
			continue
		}

		var msg JobMessage
		if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
			slog.Error("decode job message", "err", err, "raw", result[1])
			continue
		}

		if err := c.handler(ctx, msg); err != nil {
			slog.Error("handle job", "job_id", msg.JobID, "err", err)
		}
	}
}

// ensure fmt is used
var _ = fmt.Sprintf
