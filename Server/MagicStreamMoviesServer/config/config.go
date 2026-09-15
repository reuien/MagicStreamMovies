package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Port        string
	MongoURI    string
	Database    string
	SecretKey   string
	AI          AIConfig
}

type AIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load()
	cfg := Config{
		Environment: valueOrDefault("APP_ENV", "development"),
		Port:        valueOrDefault("PORT", "8080"),
		MongoURI:    os.Getenv("MONGODB_URI"),
		Database:    os.Getenv("DATABASE_NAME"),
		SecretKey:   os.Getenv("SECRET_KEY"),
		AI: AIConfig{
			APIKey:  os.Getenv("OPENAI_API_KEY"),
			BaseURL: os.Getenv("OPENAI_BASE_URL"),
			Model:   valueOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
			Timeout: durationOrDefault("AI_TIMEOUT", 30*time.Second),
		},
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	var missing []string
	for name, value := range map[string]string{
		"MONGODB_URI": c.MongoURI, "DATABASE_NAME": c.Database,
		"SECRET_KEY": c.SecretKey, "OPENAI_API_KEY": c.AI.APIKey,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %v", missing)
	}
	if c.AI.Timeout <= 0 {
		return errors.New("AI_TIMEOUT must be positive")
	}
	return nil
}

func valueOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func durationOrDefault(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return -1
	}
	return parsed
}
