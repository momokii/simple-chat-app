package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/momokii/simple-chat-app/pkg/utils"
)

var (
	SERVER_ACTIVE = true
)

func IsServerActive(c *fiber.Ctx) error {
	if !SERVER_ACTIVE {
		return utils.ResponseError(c, fiber.StatusServiceUnavailable, "Server is shutting down, your request can't be processed")
	}

	return c.Next()
}
