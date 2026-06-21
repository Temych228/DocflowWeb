package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
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
)

type App struct {
	cfg          *config.Config
	db           *pgxpool.Pool
	cache        *redis.Client
	grpcServer   *grpc.Server
	grpcListener net.Listener
	logger       *slog.Logger
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

	return &App{
		cfg:        cfg,
		db:         db,
		cache:      rdb,
		grpcServer: grpcServer,
		logger:     logger,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", a.cfg.GRPCAddress())
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", a.cfg.GRPCPort, err)
	}
	a.grpcListener = lis

	a.logger.Info("calendar-service starting", "port", a.cfg.GRPCPort)

	errCh := make(chan error, 1)
	go func() {
		if err := a.grpcServer.Serve(lis); err != nil {
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

func (a *App) Shutdown(_ context.Context) error {
	a.logger.Info("calendar-service shutting down")

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
