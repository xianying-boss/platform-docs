package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sandbox_http_requests_total",
		Help: "Total HTTP requests by method, path, and status.",
	}, []string{"method", "path", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sandbox_http_request_duration_seconds",
		Help:    "HTTP request latency by method and path.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

// Telemetry returns a Fiber middleware that records Prometheus metrics and
// structured access logs for every request.
func Telemetry() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Route().Path

		err := c.Next()

		dur := time.Since(start)
		status := c.Response().StatusCode()

		httpRequests.WithLabelValues(c.Method(), path, statusClass(status)).Inc()
		httpDuration.WithLabelValues(c.Method(), path).Observe(dur.Seconds())

		slog.Info("request",
			"method", c.Method(),
			"path", c.OriginalURL(),
			"status", status,
			"duration_ms", dur.Milliseconds(),
		)

		return err
	}
}

func statusClass(code int) string {
	switch {
	case code < 200:
		return "1xx"
	case code < 300:
		return "2xx"
	case code < 400:
		return "3xx"
	case code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}
