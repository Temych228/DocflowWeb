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
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	calendarv1 "github.com/Temych228/docflow-protos-final/calendar/v1"

	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/config"
	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/repository"
	"github.com/Temych228/DocflowWeb/services/calendar-service/internal/service"
	grpchandler "github.com/Temych228/DocflowWeb/services/calendar-service/internal/transport/grpc"
	httptransport "github.com/Temych228/DocflowWeb/services/calendar-service/internal/transport/http"
)

type App struct {
	cfg           *config.Config
	db            *pgxpool.Pool
	cache         *redis.Client
	grpcServer    *grpc.Server
	grpcListener  net.Listener
	httpServer    *http.Server
	metricsServer *http.Server
	logger        *slog.Logger
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

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warn("redis unavailable, continuing without cache", "error", err)
	}

	eventRepo := repository.NewEventRepository(db)
	svc := service.New(eventRepo)
	handler := grpchandler.New(svc)

	grpcServer := grpc.NewServer()
	calendarv1.RegisterCalendarServiceServer(grpcServer, handler)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("calendar.v1.CalendarService", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	router := gin.New()
	router.Use(gin.Recovery())

	httpHandler := httptransport.New(svc, db, rdb)
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
		cache:         rdb,
		grpcServer:    grpcServer,
		httpServer:    httpServer,
		metricsServer: metricsServer,
		logger:        logger,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	grpcLis, err := net.Listen("tcp", a.cfg.GRPCAddress())
	if err != nil {
		return fmt.Errorf("listen on grpc port %s: %w", a.cfg.GRPCPort, err)
	}
	a.grpcListener = grpcLis

	go func() {
		a.logger.Info("calendar-service gRPC listening", "port", a.cfg.GRPCPort)
		if err := a.grpcServer.Serve(grpcLis); err != nil {
			a.logger.Error("grpc server error", "error", err)
		}
	}()

	go func() {
		a.logger.Info("calendar-service HTTP listening", "addr", a.cfg.Address())
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("http server error", "error", err)
		}
	}()

	go func() {
		a.logger.Info("calendar-service metrics listening", "addr", a.cfg.MetricsAddress())
		if err := a.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("metrics server error", "error", err)
		}
	}()

	<-ctx.Done()
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("calendar-service shutting down")

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
	}

	if a.cache != nil {
		_ = a.cache.Close()
	}
	if a.db != nil {
		a.db.Close()
	}

	return nil
}
