package app

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/clients"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/config"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/messaging"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/observability"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/repository"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/scheduler"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/service"
	grpcserver "github.com/Temych228/DocflowWeb/services/notification-service/internal/transport/grpc"
	httptransport "github.com/Temych228/DocflowWeb/services/notification-service/internal/transport/http"
	notifv1 "github.com/Temych228/docflow-protos-final/notification/v1"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type App struct {
	cfg           *config.Config
	db            *pgxpool.Pool
	cache         *redis.Client
	nc            *nats.Conn
	userClient    *clients.UserClient
	publisher     messaging.Publisher
	metrics       *observability.Metrics
	grpcServer    *grpc.Server
	httpServer    *http.Server
	metricsServer *http.Server
	svc           *service.NotificationService
	scheduler     *scheduler.Runner
}

func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.PostgresDSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}

	cache := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := cache.Ping(ctx).Err(); err != nil {
		db.Close()
		_ = cache.Close()
		return nil, err
	}

	metrics := observability.NewMetrics()

	var nc *nats.Conn
	var publisher messaging.Publisher = &messaging.NoopPublisher{}
	if cfg.NATSURL != "" {
		if conn, err := nats.Connect(cfg.NATSURL); err == nil {
			nc = conn
			publisher = messaging.NewNATSPublisher(conn, cfg.LogSubject, cfg.MailJobSubject, cfg.NotificationSubject)
		}
	}

	userClient, err := clients.NewUserClient(cfg.UserServiceAddr)
	if err != nil {
		db.Close()
		_ = cache.Close()
		if nc != nil {
			_ = nc.Drain()
			nc.Close()
		}
		return nil, err
	}

	repo := repository.New(db, cache, cfg.CacheTTL, cfg.DedupTTL)
	hub := service.NewHub()
	svc := service.New(repo, publisher, userClient, hub, metrics, cfg.DedupTTL)

	if err := svc.SeedTemplates(ctx); err != nil {
		db.Close()
		_ = cache.Close()
		if nc != nil {
			_ = nc.Drain()
			nc.Close()
		}
		if userClient != nil {
			_ = userClient.Close()
		}
		return nil, err
	}

	grpcSrv := grpc.NewServer()
	notifv1.RegisterNotificationServiceServer(grpcSrv, grpcserver.New(svc))

	router := gin.New()
	router.Use(gin.Recovery())

	httpHandler := httptransport.New(svc)
	httpHandler.Register(router)

	httpServer := &http.Server{
		Addr:    cfg.Address(),
		Handler: router,
	}

	metricsServer := &http.Server{
		Addr:    cfg.MetricsAddress(),
		Handler: metrics.Handler(),
	}

	runner := scheduler.NewRunner(svc, publisher, cfg.CronEnabled)

	return &App{
		cfg:           cfg,
		db:            db,
		cache:         cache,
		nc:            nc,
		userClient:    userClient,
		publisher:     publisher,
		metrics:       metrics,
		grpcServer:    grpcSrv,
		httpServer:    httpServer,
		metricsServer: metricsServer,
		svc:           svc,
		scheduler:     runner,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	grpcLis, err := net.Listen("tcp", a.cfg.GRPCAddress())
	if err != nil {
		return err
	}

	httpLis, err := net.Listen("tcp", a.cfg.Address())
	if err != nil {
		_ = grpcLis.Close()
		return err
	}

	metricsLis, err := net.Listen("tcp", a.cfg.MetricsAddress())
	if err != nil {
		_ = grpcLis.Close()
		_ = httpLis.Close()
		return err
	}

	go func() {
		_ = a.grpcServer.Serve(grpcLis)
	}()

	go func() {
		_ = a.httpServer.Serve(httpLis)
	}()

	go func() {
		_ = a.metricsServer.Serve(metricsLis)
	}()

	if a.scheduler != nil {
		go a.scheduler.Run(ctx)
	}

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if a.scheduler != nil {
		a.scheduler.Stop()
	}
	if a.httpServer != nil {
		_ = a.httpServer.Shutdown(shutdownCtx)
	}
	if a.metricsServer != nil {
		_ = a.metricsServer.Shutdown(shutdownCtx)
	}
	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
	}
	if a.nc != nil {
		_ = a.nc.Drain()
		a.nc.Close()
	}
	if a.userClient != nil {
		_ = a.userClient.Close()
	}
	if a.cache != nil {
		_ = a.cache.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
	return nil
}
