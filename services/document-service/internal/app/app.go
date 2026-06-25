package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	documentv1 "github.com/Temych228/docflow-protos-final/document/v1"

	"github.com/Temych228/DocflowWeb/services/document-service/internal/clients"
	"github.com/Temych228/DocflowWeb/services/document-service/internal/config"
	postgres "github.com/Temych228/DocflowWeb/services/document-service/internal/repository"
	"github.com/Temych228/DocflowWeb/services/document-service/internal/service"
	grpcserver "github.com/Temych228/DocflowWeb/services/document-service/internal/transport/grpc"
	httptransport "github.com/Temych228/DocflowWeb/services/document-service/internal/transport/http"
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

	db, err := pgxpool.New(ctx, cfg.PostgresDSN())
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	documentRepo := postgres.NewDocumentRepository(db)
	historyRepo := postgres.NewHistoryRepository(db)

	notificationClient := clients.NewNoopNotificationClient(logger)
	calendarClient := clients.NewNoopCalendarClient(logger)

	svc := service.New(documentRepo, historyRepo, notificationClient, calendarClient)
	handler := grpcserver.New(svc)

	grpcServer := grpc.NewServer()
	documentv1.RegisterDocumentServiceServer(grpcServer, handler)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("document.v1.DocumentService", healthpb.HealthCheckResponse_SERVING)

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
	lis, err := net.Listen("tcp", a.cfg.GRPCAddress())
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", a.cfg.GRPCPort, err)
	}
	a.grpcListener = lis

	a.logger.Info("document-service starting", "port", a.cfg.GRPCPort)

	errCh := make(chan error, 1)
	go func() {
		if err := a.grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	httpLis, err := net.Listen("tcp", a.cfg.Address())
	if err != nil {
		_ = lis.Close()
		return fmt.Errorf("listen on app port %s: %w", a.cfg.AppPort, err)
	}

	metricsLis, err := net.Listen("tcp", a.cfg.MetricsAddress())
	if err != nil {
		_ = lis.Close()
		_ = httpLis.Close()
		return fmt.Errorf("listen on metrics port %s: %w", a.cfg.MetricsPort, err)
	}

	go func() {
		if err := a.httpServer.Serve(httpLis); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	go func() {
		if err := a.metricsServer.Serve(metricsLis); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("document-service shutting down")

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
