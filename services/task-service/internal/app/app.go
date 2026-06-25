package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	postgres "github.com/Temych228/DocflowWeb/services/task-service/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	taskv1 "github.com/Temych228/docflow-protos-final/task/v1"

	"github.com/Temych228/DocflowWeb/services/task-service/internal/clients"
	"github.com/Temych228/DocflowWeb/services/task-service/internal/config"
	"github.com/Temych228/DocflowWeb/services/task-service/internal/service"
	grpcserver "github.com/Temych228/DocflowWeb/services/task-service/internal/transport/grpc"
	httptransport "github.com/Temych228/DocflowWeb/services/task-service/internal/transport/http"
)

type App struct {
	cfg *config.Config

	db *pgxpool.Pool

	grpcServer    *grpc.Server
	grpcListener  net.Listener
	httpServer    *http.Server
	metricsServer *http.Server

	logger *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger) (*App, error) {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	taskRepo := postgres.NewTaskRepository(db)
	historyRepo := postgres.NewHistoryRepository(db)

	notificationClient := clients.NewNoopNotificationClient(logger)
	calendarClient := clients.NewNoopCalendarClient(logger)

	svc := service.New(taskRepo, historyRepo, notificationClient, calendarClient)
	handler := grpcserver.New(svc)

	grpcServer := grpc.NewServer()
	taskv1.RegisterTaskServiceServer(grpcServer, handler)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("task.v1.TaskService", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	httpHandler := httptransport.New(svc)
	httpHandler.Register(router)

	httpServer := &http.Server{
		Addr:    cfg.Address(),
		Handler: router,
	}

	metricsServer := &http.Server{
		Addr:    cfg.MetricsAddress(),
		Handler: promhttp.Handler(),
	}

	return &App{
		cfg:           cfg,
		db:            db,
		grpcServer:    grpcServer,
		httpServer:    httpServer,
		metricsServer: metricsServer,
		logger:        logger,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	grpcLis, err := net.Listen("tcp", ":"+a.cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", a.cfg.GRPCPort, err)
	}
	a.grpcListener = grpcLis

	go func() {
		a.logger.Info("task-service gRPC listening", "port", a.cfg.GRPCPort)
		if err := a.grpcServer.Serve(grpcLis); err != nil {
			a.logger.Error("grpc server error", "error", err)
		}
	}()

	go func() {
		a.logger.Info("task-service HTTP listening", "addr", a.cfg.Address())
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("http server error", "error", err)
		}
	}()

	go func() {
		a.logger.Info("task-service metrics listening", "addr", a.cfg.MetricsAddress())
		if err := a.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("metrics server error", "error", err)
		}
	}()

	<-ctx.Done()
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("task-service shutting down")

	if a.httpServer != nil {
		_ = a.httpServer.Shutdown(ctx)
	}
	if a.metricsServer != nil {
		_ = a.metricsServer.Shutdown(ctx)
	}

	stopped := make(chan struct{})
	go func() {
		a.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-ctx.Done():
		a.grpcServer.Stop()
	}

	if a.db != nil {
		a.db.Close()
	}

	return nil
}
