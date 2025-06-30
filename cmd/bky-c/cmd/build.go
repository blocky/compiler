package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/blocky/compiler/internal/bkyc"
	"github.com/blocky/compiler/internal/container"
)

var reproducible bool

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a WASM binary",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inPath := args[0]
		outPath := args[1]

		return bkyc.CompileGo(
			context.Background(),
			container.NewRuntime(slog.Default()),
			inPath,
			outPath,
			reproducible,
		)
	},
}

func init() {
	buildCmd.Flags().BoolVar(
		&reproducible,
		"reproducible",
		false,
		"ignore cached dependencies (default: false)",
	)
	rootCmd.AddCommand(buildCmd)
}
