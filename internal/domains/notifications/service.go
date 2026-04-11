package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"notificationapi.com/internal/domains/notifications/dtos"
	"notificationapi.com/internal/infrastructure/request"
	"notificationapi.com/pkg"
)

type Service struct {
	Repository Repository
	PublicKey  string
	PrivateKey string
}

func (s *Service) GenerateVAPIDKeys(ctx fiber.Ctx) error {
	vapidPublicKey, vapidPrivateKey, err := pkg.GenerateVAPIDKeys()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate VAPID keys",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"vapidPublicKey":  vapidPublicKey,
		"vapidPrivateKey": vapidPrivateKey,
	})
}

func (s *Service) CheckVAPIDKeys(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"vapidPublicKey":  s.PublicKey,
		"vapidPrivateKey": s.PrivateKey,
	})
}

func (s *Service) Subscribe(c fiber.Ctx) error {
	application, found := s.getDataFromRequest(c, "application")
	if !found {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid auth token",
		})
	}

	key, found := s.getDataFromRequest(c, "key")
	if !found {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid auth token",
		})
	}

	if application == "" || key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid auth token",
		})
	}

	req, err := request.ParseBody[dtos.RequestSubscriptionType](c)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid body",
		})
	}

	if req.User == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User must be specified",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second) // Needed for pooling
	defer cancel()

	if req.User != "" {
		_, err := s.Repository.GetUser(ctx, application, req.User)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "User not found",
			})
		}
	}

	exists, err := s.Repository.DoesEndpointExist(ctx, req.Endpoint)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal Server Error",
		})
	}

	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Device is already subscribed",
		})
	}

	err = s.Repository.Subscribe(ctx, application, req)
	if err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal Server Error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Subscription saved",
	})
}

func (s *Service) Send(ctx fiber.Ctx) error {
	application, found := s.getDataFromRequest(ctx, "application")
	if !found {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid auth token",
		})
	}

	key, found := s.getDataFromRequest(ctx, "key")
	if !found {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid auth token",
		})
	}

	fmt.Println(application)
	fmt.Println(key)

	var subscription pkg.StoredSubscription

	req, err := request.ParseBody[dtos.RequestSubscriptionType](ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid body",
		})
	}

	subscription.Endpoint = req.Endpoint
	subscription.Auth = req.Keys.Auth
	subscription.P256dh = req.Keys.P256dh
	fmt.Println(subscription)

	payload := map[string]string{
		"title": "Hola 👋",
		"body":  "Notificación enviada desde Go 🚀",
	}

	statuscode, err := pkg.SendNotification(subscription, s.PublicKey, s.PrivateKey, payload)

	if err != nil {
		fmt.Println(err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
		})
	}
	fmt.Println("Statuscode: ")
	fmt.Println(statuscode)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Notification Sent",
	})
}

func (s *Service) getDataFromRequest(ctx fiber.Ctx, fieldName string) (string, bool) {
	claims, ok := ctx.Locals("application").(jwt.MapClaims)
	if !ok {
		return "", false
	}

	field, ok := claims[fieldName].(string)
	if !ok {
		return "", false
	}

	return field, true
}
