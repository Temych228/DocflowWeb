package app

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Temych228/DocflowWeb/services/user-service/internal/config"
	"github.com/Temych228/DocflowWeb/services/user-service/internal/repository"
	"github.com/Temych228/DocflowWeb/services/user-service/internal/service"
	grpcserver "github.com/Temych228/DocflowWeb/services/user-service/internal/transport/grpc"
	httptransport "github.com/Temych228/DocflowWeb/services/user-service/internal/transport/http"
	userv1 "github.com/Temych228/docflow-protos-final/user/v1"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type App struct {
	cfg           *config.Config
	db            *pgxpool.Pool
	cache         *redis.Client
	grpcServer    *grpc.Server
	grpcListener  net.Listener
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

	repo := repository.New(db, cache, cfg.CacheTTL)
	svc := service.New(repo)

	grpcSrv := grpc.NewServer()
	userv1.RegisterUserServiceServer(grpcSrv, grpcserver.New(svc))

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
		grpcServer:    grpcSrv,
		httpServer:    httpServer,
		metricsServer: metricsServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	grpcLis, err := net.Listen("tcp", a.cfg.GRPCAddress())
	if err != nil {
		return err
	}
	a.grpcListener = grpcLis

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

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if a.httpServer != nil {
		_ = a.httpServer.Shutdown(shutdownCtx)
	}
	if a.metricsServer != nil {
		_ = a.metricsServer.Shutdown(shutdownCtx)
	}
	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
	}
	if a.grpcListener != nil {
		_ = a.grpcListener.Close()
	}
	if a.cache != nil {
		_ = a.cache.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
	return nil
}
