package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/blocky/compiler/internal/bkyc"
	"github.com/blocky/compiler/internal/container"
)

var cachePath string
var goPath string

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
			cachePath,
			goPath,
			inPath,
			outPath,
		)
	},
}

func init() {
	buildCmd.Flags().StringVar(
		&cachePath,
		"cache-path",
		"",
		"absolute path to the build cache directory",
	)
	buildCmd.Flags().StringVar(
		&goPath,
		"go-path",
		"",
		"absolute path to the go cache directory",
	)
	rootCmd.AddCommand(buildCmd)
}
