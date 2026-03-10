package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Research.MaxReportsPerSession != 50 {
		t.Errorf("expected 50, got %d", cfg.Research.MaxReportsPerSession)
	}
	if len(cfg.CodeMogger.Languages) == 0 {
		t.Errorf("expected languages to be populated")
	}
}

func TestConfigLoadSave(t *testing.T) {
	dir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)

	configPath := filepath.Join(dir, "config.json")

	// Create a custom config
	customCfg := DefaultConfig()
	customCfg.Research.MaxReportsPerSession = 100
	customCfg.System.DBPath = "/tmp/test.db"

	// Set and save
	Set(customCfg)
	// Temporarily update global path for Save
	mu.Lock()
	path = configPath
	mu.Unlock()

	if err := Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reset instance
	mu.Lock()
	instance = nil
	mu.Unlock()

	// Load
	if err := Load(configPath); err != nil {
		t.Fatalf("Load: %v", err)
	}

	loaded := Get()
	if loaded.Research.MaxReportsPerSession != 100 {
		t.Errorf("expected 100, got %d", loaded.Research.MaxReportsPerSession)
	}
	if loaded.System.DBPath != "/tmp/test.db" {
		t.Errorf("expected /tmp/test.db, got %s", loaded.System.DBPath)
	}
}

func TestConfigMerge(t *testing.T) {
	dir, err := os.MkdirTemp("", "config-merge-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)

	configPath := filepath.Join(dir, "partial.json")

	// Create a partial config file
	partial := map[string]any{
		"research": map[string]any{
			"max_reports_per_session": 75,
		},
		"system": map[string]any{
			"db_path": "~/test.db",
		},
	}
	data, _ := json.Marshal(partial)
	os.WriteFile(configPath, data, 0644)

	// Reset global state
	mu.Lock()
	instance = nil
	mu.Unlock()

	if err := Load(configPath); err != nil {
		t.Fatalf("Load: %v", err)
	}

	loaded := Get()
	if loaded.Research.MaxReportsPerSession != 75 {
		t.Errorf("expected merged 75, got %d", loaded.Research.MaxReportsPerSession)
	}
	// Default was 10
	if loaded.Research.MaxReportsPerCodebase != 10 {
		t.Errorf("expected default 10, got %d", loaded.Research.MaxReportsPerCodebase)
	}

	home, _ := os.UserHomeDir()
	expectedDBPath := filepath.Join(home, "test.db")
	if loaded.System.DBPath != expectedDBPath {
		t.Errorf("expected expanded path %s, got %s", expectedDBPath, loaded.System.DBPath)
	}
}

func TestSingleton(t *testing.T) {
	cfg1 := Get()
	cfg2 := Get()

	if cfg1 != cfg2 {
		t.Errorf("expected same instance")
	}

	newCfg := DefaultConfig()
	newCfg.Research.MaxReportsPerCodebase = 999
	Set(newCfg)

	cfg3 := Get()
	if cfg3.Research.MaxReportsPerCodebase != 999 {
		t.Errorf("expected updated value")
	}

	if !reflect.DeepEqual(cfg3, newCfg) {
		t.Errorf("expected deep equal")
	}
}
