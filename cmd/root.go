package cmd

import (
	"tg/internal/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tg",
	Short: "Telegram CLI",
}

var AppConfig config.Config
var ConfigFilePath string

func Execute() error {
	loadedConfig, loadedConfigPath, err := config.LoadOrCreate()
	if err != nil {
		return err
	}

	AppConfig = loadedConfig
	ConfigFilePath = loadedConfigPath

	return rootCmd.Execute()
}
