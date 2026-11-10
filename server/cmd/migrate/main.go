package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down" && os.Args[1] != "status") {
		return fmt.Errorf("usage: migrate up|down|status")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()
	provider, err := db.Migrations()
	if err != nil {
		return fmt.Errorf("cannot initialize migrations")
	}
	switch os.Args[1] {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return fmt.Errorf("migration up failed; consult database logs")
		}
		for _, result := range results {
			fmt.Println(result)
		}
		if len(results) == 0 {
			fmt.Println("no pending migrations")
		}
	case "down":
		result, err := provider.Down(ctx)
		if err != nil {
			return fmt.Errorf("migration down failed or no migration is applied; consult database logs")
		}
		fmt.Println(result)
	case "status":
		statuses, err := provider.Status(ctx)
		if err != nil {
			return fmt.Errorf("cannot read migration status")
		}
		for _, status := range statuses {
			fmt.Printf("%05d %s\n", status.Source.Version, status.State)
		}
	}
	return nil
}
