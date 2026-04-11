package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
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

func (s *Service) Send(c fiber.Ctx) error {
	application, found := s.getDataFromRequest(c, "application")
	if !found {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid auth token",
		})
	}

	_, found = s.getDataFromRequest(c, "key")
	if !found {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid auth token",
		})
	}

	req, err := request.ParseBody[dtos.RequestSendType](c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid body",
		})
	}

	if req.User == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User must be set",
		})
	}

	if req.Title == "" || req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Title and Message parameters must be set",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second) // Needed for pooling
	defer cancel()

	_, err = s.Repository.GetUser(ctx, application, req.User)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	payload := map[string]string{
		"title": req.Title,
		"body":  req.Message,
		"icon":  req.Icon,
	}

	var rows pgx.Rows

	if req.Tag == "" {
		rows, err = s.Repository.GetSubscriptionsByUser(ctx, application, req.User)
	} else {
		rows, err = s.Repository.GetSubscriptionsByUserAndTag(ctx, application, req.User, req.Tag)
	}

	if err != nil {
		fmt.Println("Error getting subscriptions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
		})
	}

	countSubscriptions := 0
	countRemovedSubscriptions := 0

	for rows.Next() {
		row, _ := rows.Values()
		subscription := pkg.StoredSubscription{}
		subscription.Endpoint = row[1].(string)
		subscription.P256dh = row[2].(string)
		subscription.Auth = row[3].(string)

		statuscode, err := pkg.SendNotification(subscription, s.PublicKey, s.PrivateKey, payload)

		if err != nil {
			fmt.Println("Error sending notification")
			fmt.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Internal server error",
			})
		}

		if statuscode == 410 { // Serviceworker has ben removed or unregistered and cannot send notification
			err = s.Repository.DeleteSubscription(ctx, application, subscription.Endpoint)
			if err != nil {
				fmt.Println(err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Internal server error",
				})
			}
			countRemovedSubscriptions++
		}

		countSubscriptions++
	}

	if countSubscriptions == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Couldn't find any subscriptions",
		})
	}

	if countRemovedSubscriptions == countSubscriptions {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{
			"message": "No active subscriptions where found. Inactive subscriptions have been removed",
		})
	}

	if countRemovedSubscriptions > 0 {
		return c.Status(fiber.StatusMultiStatus).JSON(fiber.Map{
			"message": "Couldn't send the notification to some subscriptions. Inactive subscriptions have been removed",
			"success": countSubscriptions,
			"failed":  countRemovedSubscriptions,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{})
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
