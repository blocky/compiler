package cmd

import (
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
)

const (
	CliName = "bky-c"
)

func StateDir() string {
	return filepath.Join(xdg.StateHome, CliName)
}

var rootCmd = &cobra.Command{
	Use:           CliName,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}
