package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredConfig(t *testing.T) {
	t.Helper()
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	t.Setenv("DATABASE_NAME", "movies")
	t.Setenv("SECRET_KEY", "test-secret")
	t.Setenv("OPENAI_API_KEY", "test-key")
}

func TestLoadAppliesDefaultsAndOverrides(t *testing.T) {
	setRequiredConfig(t)
	t.Setenv("OPENAI_MODEL", "test-model")
	t.Setenv("AI_TIMEOUT", "12s")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "8080" || cfg.AI.Model != "test-model" || cfg.AI.Timeout != 12*time.Second {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsMissingAndInvalidConfig(t *testing.T) {
	setRequiredConfig(t)
	t.Setenv("OPENAI_API_KEY", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("missing API key error = %v", err)
	}
	setRequiredConfig(t)
	t.Setenv("AI_TIMEOUT", "invalid")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted invalid AI_TIMEOUT")
	}
}
