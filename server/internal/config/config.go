package config

import "os"

type Config struct {
	Address string
	Version string
}

func Load() Config {
	return Config{
		Address: envOr("NEXUSOPS_SERVER_ADDR", ":8080"),
		Version: envOr("NEXUSOPS_VERSION", "0.1.0"),
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
