package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	ServerURL, RegistrationToken, CredentialPath, Version string
	HeartbeatInterval, MetricsInterval                    time.Duration
	MetricsQueueSize                                      int
}

func Load(version string) (Config, error) {
	c := Config{ServerURL: env("NEXUSOPS_SERVER_URL", "https://127.0.0.1:8080/api/v1"), RegistrationToken: os.Getenv("NEXUSOPS_AGENT_REGISTRATION_TOKEN"), CredentialPath: env("NEXUSOPS_AGENT_CREDENTIAL_FILE", "/var/lib/nexusops-agent/credential"), Version: version}
	v := env("NEXUSOPS_AGENT_HEARTBEAT_INTERVAL", "30s")
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return c, errors.New("NEXUSOPS_AGENT_HEARTBEAT_INTERVAL must be positive duration")
	}
	c.HeartbeatInterval = d
	metricsInterval, err := time.ParseDuration(env("NEXUSOPS_AGENT_METRICS_INTERVAL", "30s"))
	if err != nil || metricsInterval < 10*time.Second || metricsInterval > 5*time.Minute {
		return c, errors.New("NEXUSOPS_AGENT_METRICS_INTERVAL must be between 10s and 5m")
	}
	c.MetricsInterval = metricsInterval
	queueSize := 4
	if raw := os.Getenv("NEXUSOPS_AGENT_METRICS_QUEUE_SIZE"); raw != "" {
		if _, err := fmt.Sscanf(raw, "%d", &queueSize); err != nil || queueSize < 1 || queueSize > 32 {
			return c, errors.New("NEXUSOPS_AGENT_METRICS_QUEUE_SIZE must be between 1 and 32")
		}
	}
	c.MetricsQueueSize = queueSize
	if c.ServerURL == "" {
		return c, errors.New("NEXUSOPS_SERVER_URL is required")
	}
	return c, nil
}
func env(k, f string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return f
}
func (c Config) Credential() (string, error) {
	b, err := os.ReadFile(c.CredentialPath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (c Config) SaveCredential(v string) error {
	if v == "" {
		return errors.New("empty credential")
	}
	if err := os.MkdirAll(filepath.Dir(c.CredentialPath), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(c.CredentialPath, []byte(v), 0600); err != nil {
		return fmt.Errorf("save credential: %w", err)
	}
	_ = os.Chmod(c.CredentialPath, 0600)
	return nil
}
