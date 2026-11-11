package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/740297616/nexusops/agent/internal/client"
	"github.com/740297616/nexusops/agent/internal/collector"
	"github.com/740297616/nexusops/agent/internal/config"
)

var version = "dev"

func main() {
	cfg, err := config.Load(version)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	info := collector.Collect()
	info.AgentVersion = cfg.Version
	c := client.New(cfg)
	if cfg.RegistrationToken != "" {
		credential, err := c.Register(ctx, info)
		if err != nil {
			log.Printf("agent registration failed: %v", err)
		} else if err := cfg.SaveCredential(credential); err != nil {
			log.Fatal(err)
		}
	}
	credential, err := cfg.Credential()
	if err != nil {
		log.Fatal(err)
	}
	if credential == "" {
		log.Fatal("agent credential is required")
	}
	c.SetCredential(credential)
	if err := c.Report(ctx, info); err != nil {
		log.Printf("initial report failed: %v", err)
	}
	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.Heartbeat(ctx, info); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
