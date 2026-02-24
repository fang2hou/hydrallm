package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	origCfgFile := cfgFile
	origHome := os.Getenv("HOME")
	defer func() {
		cfgFile = origCfgFile
		if origHome != "" {
			_ = os.Setenv("HOME", origHome)
		} else {
			_ = os.Unsetenv("HOME")
		}
	}()

	t.Run("custom config file", func(t *testing.T) {
		cfgFile = "/custom/config/path.toml"
		if got := getConfigPath(); got != "/custom/config/path.toml" {
			t.Errorf("expected custom config path, got %q", got)
		}
	})

	t.Run("default config path", func(t *testing.T) {
		cfgFile = ""
		_ = os.Setenv("HOME", "/dummy/home")
		expected := filepath.Join("/dummy/home", ".config", "hydrallm", "config.toml")
		if got := getConfigPath(); got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})
}

func TestNewEditCmd(t *testing.T) {
	cmd := newEditCmd()
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Use != "edit" {
		t.Errorf("expected Use 'edit', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
	if cmd.Run == nil {
		t.Error("expected Run function")
	}
}

func TestDefaultConfigTemplate(t *testing.T) {
	// Verify the embedded config template is not empty
	if defaultConfigTemplate == "" {
		t.Error("defaultConfigTemplate should not be empty")
	}
}
