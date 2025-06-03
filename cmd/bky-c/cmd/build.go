package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/blocky/bkyc/internal/bkyc"
	"github.com/blocky/bkyc/internal/container"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a WASM binary",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return bkyc.CompileGo(
			context.Background(),
			container.NewRuntime(slog.Default()),
			args[0],
			args[1],
		)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
