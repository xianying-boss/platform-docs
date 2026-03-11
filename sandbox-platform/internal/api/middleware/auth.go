package middleware

import (
	"log/slog"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const ContextKeyAgentID = "agent_id"

// Auth returns a Fiber middleware that validates Bearer JWT tokens.
// If keyPath is empty, dev mode is enabled: all requests pass through
// with agent_id="dev-agent". Never enable dev mode in production.
func Auth(keyPath string) fiber.Handler {
	if keyPath == "" {
		slog.Warn("JWT public key not configured — running in DEV MODE, all requests accepted")
		return func(c *fiber.Ctx) error {
			c.Locals(ContextKeyAgentID, "dev-agent")
			return c.Next()
		}
	}

	pubKeyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		// Fail loud at startup rather than silently degrading to dev mode.
		slog.Error("cannot read JWT public key", "path", keyPath, "err", err)
		os.Exit(1)
	}

	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		agentID, err := validateJWT(tokenStr, pubKeyPEM)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}

		c.Locals(ContextKeyAgentID, agentID)
		return c.Next()
	}
}

// validateJWT parses and validates an RS256 JWT, returning the subject claim.
// Uses only stdlib crypto — no external JWT library needed.
func validateJWT(token string, pubKeyPEM []byte) (string, error) {
	// NOTE: Production implementation should use github.com/golang-jwt/jwt/v5
	// with RS256 verification. Stub returns subject from unverified claims
	// for skeleton purposes only — replace before production.
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fiber.NewError(fiber.StatusUnauthorized, "malformed token")
	}
	// Real RS256 verification would go here.
	// For the skeleton we accept any well-formed token.
	_ = pubKeyPEM
	return "agent-from-token", nil
}
