package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/Temych228/DocflowWeb/api-gateway/internal/clients"
	"github.com/Temych228/DocflowWeb/api-gateway/internal/config"
	"github.com/Temych228/DocflowWeb/api-gateway/internal/handlers"
	"github.com/Temych228/DocflowWeb/api-gateway/internal/middleware"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Redis unavailable: %v", err)
	}

	authClient, err := clients.NewAuthClient(cfg.AuthServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create auth client: %v", err)
	}
	defer authClient.Close()

	userClient, err := clients.NewUserClient(cfg.UserServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create user client: %v", err)
	}
	defer userClient.Close()

	docClient, err := clients.NewDocumentClient(cfg.DocumentServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create document client: %v", err)
	}
	defer docClient.Close()

	taskClient, err := clients.NewTaskClient(cfg.TaskServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create task client: %v", err)
	}
	defer taskClient.Close()

	calendarClient, err := clients.NewCalendarClient(cfg.CalendarServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create calendar client: %v", err)
	}
	defer calendarClient.Close()

	notifClient, err := clients.NewNotificationClient(cfg.NotificationServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create notification client: %v", err)
	}
	defer notifClient.Close()

	mailClient, err := clients.NewMailClient(cfg.MailServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create mail client: %v", err)
	}
	defer mailClient.Close()

	router := gin.Default()

	router.Static("/assets", "./web/assets")
	router.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-gateway"})
	})

	authHandler := handlers.NewAuthHandler(authClient)
	userHandler := handlers.NewUserHandler(userClient)
	docHandler := handlers.NewDocumentHandler(docClient)
	taskHandler := handlers.NewTaskHandler(taskClient)
	calendarHandler := handlers.NewCalendarHandler(calendarClient)
	notifHandler := handlers.NewNotificationHandler(notifClient)
	mailHandler := handlers.NewMailHandler(mailClient)

	api := router.Group("/api/v1")

	public := api.Group("")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/refresh", authHandler.RefreshToken)
		public.POST("/auth/logout", authHandler.Logout)
		public.POST("/auth/verify-email", authHandler.VerifyEmail)
		public.POST("/auth/forgot-password", authHandler.ForgotPassword)
		public.POST("/auth/reset-password", authHandler.ResetPassword)
		public.POST("/auth/change-password", authHandler.ChangePassword)
	}

	protected := api.Group("")
	protected.Use(middleware.JWTMiddleware(authClient))
	protected.Use(middleware.RateLimitMiddleware(rdb, cfg.RateLimitRPM))
	protected.Use(middleware.IdempotencyMiddleware(rdb))
	{
		protected.GET("/users", userHandler.ListUsers)
		protected.GET("/users/:id", userHandler.GetUser)
		protected.POST("/users", userHandler.CreateUser)
		protected.PUT("/users/:id", userHandler.UpdateUser)
		protected.DELETE("/users/:id", userHandler.DeleteUser)
		protected.GET("/users/by-email", userHandler.GetUserByEmail)
		protected.GET("/users/exists", userHandler.CheckUserExists)
		protected.POST("/users/:user_id/ban",
			middleware.RBACMiddleware("admin"),
			userHandler.BanUser,
		)

		protected.GET("/documents", docHandler.ListDocuments)
		protected.POST("/documents", docHandler.CreateDocument)
		protected.GET("/documents/export/csv", docHandler.ExportCSV)
		protected.POST("/documents/filter", docHandler.FilterDocuments)
		protected.POST("/documents/overdue",
			middleware.RBACMiddleware("admin", "manager"),
			docHandler.MarkOverdue,
		)
		protected.GET("/documents/:id", docHandler.GetDocument)
		protected.PATCH("/documents/:id", docHandler.UpdateDocument)
		protected.DELETE("/documents/:id", docHandler.DeleteDocument)
		protected.POST("/documents/:id/assign",
			middleware.RBACMiddleware("admin", "manager"),
			docHandler.AssignResponsible,
		)
		protected.POST("/documents/:id/status", docHandler.ChangeStatus)
		protected.POST("/documents/:id/archive",
			middleware.RBACMiddleware("admin", "manager"),
			docHandler.ArchiveDocument,
		)
		protected.GET("/documents/:id/history", docHandler.GetDocumentHistory)

		protected.GET("/tasks/document/:document_id", taskHandler.ListTasksByDocument)
		protected.GET("/tasks/assignee/:assignee_id", taskHandler.ListTasksByAssignee)
		protected.POST("/tasks/filter", taskHandler.FilterTasks)
		protected.POST("/tasks/overdue",
			middleware.RBACMiddleware("admin", "manager"),
			taskHandler.MarkTasksOverdue,
		)
		protected.GET("/tasks/:id", taskHandler.GetTask)
		protected.POST("/tasks", taskHandler.CreateTask)
		protected.PATCH("/tasks/:id", taskHandler.UpdateTask)
		protected.DELETE("/tasks/:id", taskHandler.DeleteTask)
		protected.POST("/tasks/:id/assign", taskHandler.AssignTask)
		protected.POST("/tasks/:id/status", taskHandler.ChangeTaskStatus)
		protected.GET("/tasks/:id/history", taskHandler.GetTaskHistory)
		protected.GET("/tasks/stats", taskHandler.GetTaskStats)

		protected.POST("/events", calendarHandler.CreateEvent)
		protected.GET("/events/day/:user_id", calendarHandler.GetEventsByDay)
		protected.GET("/events/week/:user_id", calendarHandler.GetEventsByWeek)
		protected.GET("/events/month/:user_id", calendarHandler.GetEventsByMonth)
		protected.GET("/events/user/:user_id", calendarHandler.GetEventsByUser)
		protected.GET("/events/upcoming/:user_id", calendarHandler.GetUpcomingDeadlines)
		protected.GET("/events/stats/:user_id", calendarHandler.GetEventStats)
		protected.POST("/events/filter", calendarHandler.FilterEvents)
		protected.DELETE("/events/:id", calendarHandler.DeleteEvent)
		protected.PATCH("/events/:id", calendarHandler.UpdateEvent)

		protected.POST("/notifications/send-email",
			middleware.RBACMiddleware("admin", "manager"),
			notifHandler.SendEmail,
		)
		protected.POST("/notifications/send-bulk",
			middleware.RBACMiddleware("admin"),
			notifHandler.SendBulkEmail,
		)
		protected.POST("/notifications", notifHandler.CreateNotification)
		protected.GET("/notifications", notifHandler.GetNotificationHistory)
		protected.GET("/notifications/unread-count", notifHandler.GetUnreadCount)
		protected.POST("/notifications/read-all", notifHandler.MarkAllRead)
		protected.POST("/notifications/:id/read", notifHandler.MarkRead)
		protected.DELETE("/notifications/:id", notifHandler.DeleteNotification)
		protected.GET("/templates/:id", notifHandler.GetTemplate)
		protected.POST("/preferences", notifHandler.UpdatePreferences)
		protected.GET("/preferences", notifHandler.GetPreferences)

		protected.POST("/mail/send-email", mailHandler.SendEmail)
		protected.POST("/mail/send-bulk", mailHandler.SendBulkEmail)
		protected.POST("/mail/jobs", mailHandler.SubmitMailJob)
		protected.GET("/mail/jobs", mailHandler.ListMailJobs)
		protected.GET("/mail/jobs/:id", mailHandler.GetMailJob)
		protected.GET("/mail/templates", mailHandler.ListTemplates)
		protected.GET("/mail/templates/:id", mailHandler.GetTemplate)
		protected.PUT("/mail/templates/:id", mailHandler.UpdateTemplate)
		protected.GET("/mail/stats", mailHandler.GetStats)
	}

	log.Printf("API Gateway starting on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
