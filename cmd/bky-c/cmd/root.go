package cmd

import (
	"github.com/spf13/cobra"
)

const (
	cliName = "bky-c"
)

var rootCmd = &cobra.Command{
	Use:          cliName,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}
