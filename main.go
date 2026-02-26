package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func main() {
	cmd := &cobra.Command{
		Use:   "hydrallm",
		Short: "LLM API proxy with automatic retry and fallback",
		Run:   runServe,
	}

	cobra.OnInitialize(initConfig)
	cmd.PersistentFlags().
		StringVarP(&cfgFile, "config", "c", "", "config file (default is ~/.config/hydrallm/config.toml)")
	cmd.PersistentFlags().StringP("log-level", "l", "", "log level (debug, info, warn, error)")

	_ = viper.BindPFlag("log.level", cmd.PersistentFlags().Lookup("log-level"))

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newEditCmd())

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Fatalf("failed to get home directory: %v", err)
		}
		viper.AddConfigPath(filepath.Join(home, ".config", "hydrallm"))
		viper.SetConfigType("toml")
		viper.SetConfigName("config")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			logger.Fatalf("failed to read config: %v", err)
		}
	}
}
