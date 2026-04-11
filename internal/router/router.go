package router

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/jackc/pgx/v5/pgxpool"
	"notificationapi.com/internal/config"
	"notificationapi.com/internal/domains/auth"
	"notificationapi.com/internal/domains/notifications"
	"notificationapi.com/internal/infrastructure/database"
	"notificationapi.com/internal/infrastructure/domaincreator"
	"notificationapi.com/internal/middlewares"
)

type Router struct {
	app           *fiber.App
	configuration *config.Config
	db            *pgxpool.Pool
}

func (r *Router) Initialize() {
	r.setDefaultVariables()
	r.establishDBConnectionPool()
	r.createRouter()
	err := r.app.Listen(":" + r.configuration.Application.Port)
	if err != nil {
		log.Panicf("Failed to start server: %v", err)
	}
}

func (r *Router) setDefaultVariables() {
	db_port, _ := strconv.Atoi(os.Getenv("DB_PORT"))

	r.configuration = &config.Config{}
	r.configuration.Application.MaxRequestsPerMinute = 1000
	r.configuration.Application.Port = "3000"
	r.configuration.Application.VAPIDPrivateKey = os.Getenv("VAPIDPRIVATEKEY")
	r.configuration.Application.VAPIDPubliKey = os.Getenv("VAPIDPUBLICKEY")

	r.configuration.Application.SecretJWT = os.Getenv("JWT_SECRET")
	r.configuration.Application.HmacKey = os.Getenv("JWT_HMACKEY")
	r.configuration.Application.EncryptionKey = os.Getenv("JWT_ENCRIPTIONKEY")

	r.configuration.Database.Host = os.Getenv("DB_HOST")
	r.configuration.Database.Name = os.Getenv("DB_NAME")
	r.configuration.Database.Username = os.Getenv("DB_USER")
	r.configuration.Database.Password = os.Getenv("DB_PASSWORD")
	r.configuration.Database.Port = db_port
	r.configuration.Database.SSLMode = os.Getenv("DB_SSLMODE")

	r.app = fiber.New(fiber.Config{
		Immutable: true,
	})

	r.setRateLimiter()
}

func (r *Router) establishDBConnectionPool() {
	fmt.Println("Establishing DB connection pool...")
	db, err := database.NewPostgres(*r.configuration)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("Connection pool established succesfully...")
	r.db = db.Pool
}

func (r *Router) setRateLimiter() {
	if r.configuration.Application.MaxRequestsPerMinute == 0 {
		return
	}
	r.app.Use(limiter.New(limiter.Config{
		Max:        r.configuration.Application.MaxRequestsPerMinute,
		Expiration: 1 * time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests",
			})
		},
	}))
}

func (r *Router) createRouter() {
	r.setAuthRoutes()
	r.setWebPushRoutes()
}

func (r *Router) setWebPushRoutes() {
	repository := domaincreator.Create[notifications.Repository]()
	repository.DB = r.db

	service := domaincreator.Create[notifications.Service]()
	service.PrivateKey = r.configuration.Application.VAPIDPrivateKey
	service.PublicKey = r.configuration.Application.VAPIDPubliKey
	service.Repository = *repository

	webpushGroup := r.app.Group("/notifications",
		middlewares.ContentTypeAllowed("application/json"),
		middlewares.JWTMiddleware([]byte(r.configuration.Application.SecretJWT)),
	)

	webpushGroup.Post("/subscribe", service.Subscribe)
	webpushGroup.Post("/send", service.Send)
}

func (r *Router) setAuthRoutes() {
	repository := domaincreator.Create[auth.Repository]()
	repository.DB = r.db

	service := domaincreator.Create[auth.Service]()
	service.Repository = *repository
	service.Secret = r.configuration.Application.SecretJWT

	authGroup := r.app.Group("/auth")

	authGroup.Post("/",
		middlewares.ContentTypeAllowed("application/json"),
		service.Login,
	)
}
