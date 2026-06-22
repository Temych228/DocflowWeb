package app

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/config"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/messaging"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/observability"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/repository"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/sender"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/service"
	grpctransport "github.com/Temych228/DocflowWeb/services/mail-service/internal/transport/grpc"
	httptransport "github.com/Temych228/DocflowWeb/services/mail-service/internal/transport/http"

	mailv1 "github.com/Temych228/docflow-protos-final/mail/v1"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type App struct {
	cfg *config.Config

	db    *pgxpool.Pool
	cache *redis.Client
	nc    *nats.Conn

	service *service.MailService

	grpcServer    *grpc.Server
	httpServer    *http.Server
	metricsServer *http.Server
}

func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.PostgresDSN())
	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	cache := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := cache.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	var nc *nats.Conn

	if cfg.NATSURL != "" {
		nc, _ = nats.Connect(cfg.NATSURL)
	}

	var publisher messaging.Publisher = &messaging.NoopPublisher{}
	if nc != nil {
		publisher = messaging.NewNATSPublisher(nc, cfg.LogSubject)
	}

	mailer := sender.NewSMTPMailer(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUsername,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
		cfg.SMTPUseTLS,
		cfg.SMTPUseStartTLS,
		cfg.SMTPSkipVerify,
		cfg.SMTPTimeout,
	)

	metrics := observability.NewMetrics()

	repo := repository.New(db, cache, cfg.CacheTTL, cfg.DedupTTL)

	svc := service.New(
		repo,
		publisher,
		mailer,
		metrics,
		cfg.SMTPFrom,
		cfg.DedupTTL,
	)

	grpcSrv := grpc.NewServer()

	grpcHandler := grpctransport.New(svc)

	mailv1.RegisterMailServiceServer(
		grpcSrv,
		grpcHandler,
	)

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
		Handler: promhttp.Handler(),
	}

	return &App{
		cfg:           cfg,
		db:            db,
		cache:         cache,
		nc:            nc,
		service:       svc,
		grpcServer:    grpcSrv,
		httpServer:    httpServer,
		metricsServer: metricsServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	grpcListener, err := net.Listen(
		"tcp",
		a.cfg.GRPCAddress(),
	)
	if err != nil {
		return err
	}

	httpListener, err := net.Listen(
		"tcp",
		a.cfg.Address(),
	)
	if err != nil {
		return err
	}

	metricsListener, err := net.Listen(
		"tcp",
		a.cfg.MetricsAddress(),
	)
	if err != nil {
		return err
	}

	go func() {
		_ = a.grpcServer.Serve(grpcListener)
	}()

	go func() {
		_ = a.httpServer.Serve(httpListener)
	}()

	go func() {
		_ = a.metricsServer.Serve(metricsListener)
	}()

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	timeout, cancel := context.WithTimeout(
		ctx,
		10*time.Second,
	)
	defer cancel()

	if a.httpServer != nil {
		_ = a.httpServer.Shutdown(timeout)
	}

	if a.metricsServer != nil {
		_ = a.metricsServer.Shutdown(timeout)
	}

	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
	}

	if a.nc != nil {
		_ = a.nc.Drain()
		a.nc.Close()
	}

	if a.cache != nil {
		_ = a.cache.Close()
	}

	if a.db != nil {
		a.db.Close()
	}

	return nil
}
