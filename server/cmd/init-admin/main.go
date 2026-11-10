package main

import (
	"context"
	"log"
	"os"

	"github.com/740297616/nexusops/server/internal/auth"
	"github.com/740297616/nexusops/server/internal/config"
	"github.com/740297616/nexusops/server/internal/database"
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
	service, err := auth.NewService(auth.NewRepository(db.GORM()))
	if err != nil {
		return err
	}
	if err := service.InitializeAdmin(context.Background(), os.Getenv("NEXUSOPS_ADMIN_USER"), os.Getenv("NEXUSOPS_ADMIN_DISPLAY_NAME"), os.Getenv("NEXUSOPS_ADMIN_PASSWORD")); err != nil {
		return err
	}
	log.Print("administrator initialized")
	return nil
}
