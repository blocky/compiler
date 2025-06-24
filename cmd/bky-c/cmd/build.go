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

func cleanUp(s *state.State, cleaner state.Cleaner, log state.Logger) error {
	if err := s.Finalize(cleaner); err != nil {
		return fmt.Errorf("failed to clean-up state: %w", err)
	}
	if err := state.CleanupStale(StateDir(), cleaner, log); err != nil {
		return fmt.Errorf("failed to clean up stale state: %w", err)
	}
	return nil
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a WASM binary",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := slog.Default()
		s, err := state.Init(StateDir(), os.Getpid(), log)
		if err != nil {
			return fmt.Errorf("initializing state: %w", err)
		}
		runtime := container.NewRuntime(s, log)
		defer func() {
			if err := cleanUp(s, runtime, log); err != nil {
				log.Warn("clean-up failed:", "err", err.Error())
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
