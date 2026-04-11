package middlewares

import (
	"github.com/gofiber/fiber/v3"
	"notificationapi.com/internal/infrastructure/request"
)

func JWTMiddleware(secret []byte) fiber.Handler {
	return func(c fiber.Ctx) error {
		claims, ok := request.ValidateJWT(c, secret)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Valid bearer token must be specified",
			})
		}
		c.Locals("application", claims)
		return c.Next()
	}
}
