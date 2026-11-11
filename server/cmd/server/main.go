package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/740297616/nexusops/server/internal/api"
	"github.com/740297616/nexusops/server/internal/auth"
	"github.com/740297616/nexusops/server/internal/config"
	"github.com/740297616/nexusops/server/internal/database"
	"github.com/740297616/nexusops/server/internal/rbac"
	"github.com/740297616/nexusops/server/internal/system"
	"github.com/740297616/nexusops/server/internal/serverasset"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(context.Background(), cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()
	authService, err := auth.NewServiceWithOptions(auth.NewRepository(db.GORM()), auth.Options{TrustedProxyCIDRs: cfg.Security.TrustedProxyCIDRs, RateLimitIPMax: cfg.Security.RateLimitIPMax, RateLimitAccountMax: cfg.Security.RateLimitAccountMax, RateLimitWindow: cfg.Security.RateLimitWindow})
	if err != nil {
		return err
	}
	router := api.NewRouter(cfg.Version, db.Ping)
	rbacService := rbac.NewService(rbac.NewRepository(db.GORM()))
	auth.RegisterRoutes(router, authService, rbacService.Effective)
	rbac.RegisterRoutes(router, authService, rbacService)
	system.RegisterRoutes(router, authService, rbacService, system.NewService(system.NewRepository(db.GORM())))
	assetService := serverasset.NewService(serverasset.NewRepository(db.GORM()))
	serverasset.RegisterRoutes(router, authService, rbacService, assetService, cfg.Security.AgentRegistrationTTL)
	serverasset.RegisterAgentRoutes(router, assetService)
	authService.StartCleanup(context.Background(), cfg.Security.CleanupInterval, cfg.Security.SessionRetention, cfg.Security.AuditRetention)
	go func() {
		ticker := time.NewTicker(cfg.Security.AgentHeartbeatTimeout / 2)
		defer ticker.Stop()
		for range ticker.C { if err := assetService.MarkOffline(context.Background(), cfg.Security.AgentHeartbeatTimeout); err != nil { log.Printf("agent status update failed: %v", err) } }
	}()
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("starting server on %s", cfg.Address)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()
	select {
	case err := <-serveErr:
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return err
		}
		return nil
	}
}
