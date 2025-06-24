package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/blocky/compiler/internal/bkyc"
	"github.com/blocky/compiler/internal/container"
	"github.com/blocky/compiler/internal/state"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a WASM binary",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		s, err := state.Init(StateDir(), os.Getpid(), logger)
		if err != nil {
			return fmt.Errorf("initializing state: %w", err)
		}
		runtime := container.NewRuntime(s, logger)

		defer func() {
			err := s.Finalize(runtime)
			if err != nil {
				logger.Warn("Failed to clean-up state", "err", err.Error())
			}
		}()

		return bkyc.CompileGo(
			context.Background(),
			runtime,
			args[0],
			args[1],
		)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
