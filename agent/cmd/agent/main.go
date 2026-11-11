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
	"github.com/740297616/nexusops/agent/internal/metrics"
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
	go runHeartbeat(ctx, c, info, cfg.HeartbeatInterval)
	go runMetrics(ctx, c, cfg)
	<-ctx.Done()
}

func runHeartbeat(ctx context.Context, c *client.Client, info collector.Info, interval time.Duration) {
	ticker := time.NewTicker(interval)
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

func runMetrics(ctx context.Context, c *client.Client, cfg config.Config) {
	collector := metrics.NewCollector()
	queue := make(chan metrics.Batch, cfg.MetricsQueueSize)
	go func() {
		ticker := time.NewTicker(cfg.MetricsInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				batch, err := collector.Collect(ctx)
				if err != nil {
					log.Printf("metrics collection failed: %v", err)
				}
				if len(batch.Samples) == 0 {
					continue
				}
				select {
				case queue <- batch:
				default:
					select {
					case <-queue:
					default:
					}
					queue <- batch
				}
			case <-ctx.Done():
				close(queue)
				return
			}
		}
	}()
	for batch := range queue {
		if err := c.ReportMetrics(ctx, batch); err != nil {
			log.Printf("metrics report failed: %v", err)
		}
	}
}
