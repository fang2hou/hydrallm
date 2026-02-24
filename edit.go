package main

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

//go:embed config.example.toml
var defaultConfigTemplate string

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open config file in $EDITOR",
		Run:   runEdit,
	}
}

func runEdit(_ *cobra.Command, _ []string) {
	configPath := getConfigPath()

	// Create default config if not exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
			logger.Fatalf("failed to create config directory: %v", err)
		}
		if err := os.WriteFile(configPath, []byte(defaultConfigTemplate), 0o644); err != nil {
			logger.Fatalf("failed to create default config: %v", err)
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = detectEditor()
	}

	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logger.Fatalf("failed to open editor: %v", err)
	}
}

func detectEditor() string {
	editors := []string{"code", "zed", "neovim", "vim", "vi"}
	if runtime.GOOS == "windows" {
		editors = append(editors, "notepad")
	}

	for _, editor := range editors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	logger.Fatalf(
		"no editor found. Please set EDITOR environment variable: EDITOR=<editor> %s edit",
		"hydrallm",
	)
	return ""
}

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Fatalf("failed to get home directory: %v", err)
	}
	return filepath.Join(home, ".config", "hydrallm", "config.toml")
}
