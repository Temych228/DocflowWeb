package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	authv1 "github.com/Temych228/docflow-protos-final/auth/v1"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/Temych228/DocflowWeb/services/auth-service/internal/clients"
	"github.com/Temych228/DocflowWeb/services/auth-service/internal/config"
	"github.com/Temych228/DocflowWeb/services/auth-service/internal/repository"
	"github.com/Temych228/DocflowWeb/services/auth-service/internal/service"
	grpcserver "github.com/Temych228/DocflowWeb/services/auth-service/internal/transport/grpc"
)

type App struct {
	cfg          *config.Config
	db           *pgxpool.Pool
	cache        *redis.Client
	grpcServer   *grpc.Server
	grpcListener net.Listener
	httpServer   *http.Server
}

func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	cache := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := cache.Ping(ctx).Err(); err != nil {
		db.Close()
		_ = cache.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	var notifClient clients.NotificationClient
	if nc, err := clients.NewNotificationClient(cfg.NotificationServiceAddr); err != nil {
		log.Printf("[WARN] notification-service unavailable (%s): %v — email sending disabled", cfg.NotificationServiceAddr, err)
	} else {
		notifClient = nc
	}

	var userClient clients.UserClient
	if uc, err := clients.NewUserClient(cfg.UserServiceAddr); err != nil {
		log.Printf("[WARN] user-service unavailable (%s): %v — user sync disabled", cfg.UserServiceAddr, err)
	} else {
		userClient = uc
	}

	repo := repository.New(db, cache)
	svc := service.New(cfg, repo, notifClient, userClient)

	grpcSrv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcSrv, grpcserver.New(svc))
	reflection.Register(grpcSrv)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	httpServer := &http.Server{
		Addr:         cfg.Address(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &App{
		cfg:        cfg,
		db:         db,
		cache:      cache,
		grpcServer: grpcSrv,
		httpServer: httpServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	grpcLis, err := net.Listen("tcp", a.cfg.GRPCAddress())
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}
	a.grpcListener = grpcLis

	go func() {
		log.Printf("[INFO] auth-service gRPC listening on %s", a.cfg.GRPCAddress())
		if err := a.grpcServer.Serve(grpcLis); err != nil {
			log.Printf("[ERROR] grpc server stopped: %v", err)
		}
	}()

	go func() {
		log.Printf("[INFO] auth-service HTTP (metrics) listening on %s", a.cfg.Address())
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ERROR] http server stopped: %v", err)
		}
	}()

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if a.httpServer != nil {
		_ = a.httpServer.Shutdown(shutdownCtx)
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

	log.Println("[INFO] auth-service shutdown complete")
	return nil
}
