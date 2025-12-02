package main

import (
	"log"
	"treasureHunt/config"
	"treasureHunt/database"
	"treasureHunt/handlers"
	"treasureHunt/middleware"
	"treasureHunt/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Connect to database
	database.ConnectDB()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     config.AppConfig.FrontendURL,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	}))

	// Initialize WebSocket hub
	wsHub := handlers.NewHub()
	go wsHub.Run() // Start hub in a goroutine

	// Initialize handlers
	authHandler := handlers.NewAuthHandler()
	imageHandler := handlers.NewImageHandler()

	// Initialize repositories
	challengeRepo := repository.NewChallengeRepository(database.DB)
	friendRepo := repository.NewFriendRepository(database.DB)
	battleRepo := repository.NewBattleRepository(database.DB)
	notificationRepo := repository.NewNotificationRepository(database.DB)

	// Initialize handlers
	challengeHandler := handlers.NewChallengeHandler(challengeRepo, imageHandler.OpenAIClient)
	friendHandler := handlers.NewFriendHandler(friendRepo, notificationRepo, wsHub)
	battleHandler := handlers.NewBattleHandler(battleRepo, friendRepo, notificationRepo, challengeRepo, wsHub)
	wsHandler := handlers.NewWebSocketHandler(wsHub)

	// Set battle repo in challenge handler for syncing progress
	challengeHandler.SetBattleRepo(battleRepo)

	// Public routes
	api := app.Group("/api")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/signup", authHandler.Signup)
	auth.Post("/login", authHandler.Login)
	auth.Get("/google", authHandler.GoogleLogin)
	auth.Get("/google/callback", authHandler.GoogleCallback)

	// Protected routes
	protected := api.Group("", middleware.AuthRequired())
	protected.Get("/auth/me", authHandler.GetMe)
	protected.Post("/getImageContents", imageHandler.GetImageContents)

	// WebSocket route (protected) - outside of /api group to handle auth via query param
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			// Extract token from query parameter for WebSocket
			token := c.Query("token")
			if token != "" {
				// Set it as Authorization header for middleware
				c.Request().Header.Set("Authorization", "Bearer "+token)
			}
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", middleware.AuthRequired(), websocket.New(func(c *websocket.Conn) {
		wsHandler.HandleWebSocket(c)
	}))

	// Challenge routes
	challenges := protected.Group("/challenges")
	challenges.Post("/start", challengeHandler.StartChallenge)
	challenges.Get("", challengeHandler.GetUserChallenges)
	challenges.Get("/:challengeId", challengeHandler.GetChallenge)
	challenges.Post("/:challengeId/:itemId", challengeHandler.VerifyItem)

	// Friend routes
	friends := protected.Group("/friends")
	friends.Post("/request", friendHandler.SendFriendRequest)
	friends.Get("/requests", friendHandler.GetFriendRequests)
	friends.Post("/requests/:requestId/accept", friendHandler.AcceptFriendRequest)
	friends.Post("/requests/:requestId/reject", friendHandler.RejectFriendRequest)
	friends.Get("", friendHandler.GetFriends)
	friends.Post("/online", friendHandler.UpdateOnlineStatus)
	friends.Get("/search", friendHandler.SearchUserByEmail)

	// Notification routes
	notifications := protected.Group("/notifications")
	notifications.Get("", friendHandler.GetNotifications)
	notifications.Post("/:notificationId/read", friendHandler.MarkNotificationRead)
	notifications.Post("/read-all", friendHandler.MarkAllNotificationsRead)
	notifications.Get("/unread-count", friendHandler.GetUnreadCount)

	// Battle routes
	battles := protected.Group("/battles")
	battles.Post("", battleHandler.CreateBattle)
	battles.Get("", battleHandler.GetUserBattles)
	battles.Get("/active", battleHandler.GetActiveBattles)
	battles.Get("/:battleId", battleHandler.GetBattle)
	battles.Post("/:battleId/accept", battleHandler.AcceptBattle)
	battles.Post("/:battleId/decline", battleHandler.DeclineBattle)
	battles.Post("/:battleId/join", battleHandler.JoinBattle)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	// Start server
	port := config.AppConfig.ServerPort
	log.Printf("Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
