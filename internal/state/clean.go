package state

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
)

type Cleaner interface {
	CleanUp(context.Context, string) error
}

func ProcRunning(pid int) (bool, error) {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("finding proc '%d: %w'", pid, err)
	}
	return p.Signal(syscall.Signal(0)) == nil, nil
}

func CleanupStale(dirPath string, cleaner Cleaner, log *slog.Logger) error {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("reading app state dir: %w", err)
	}
	for _, e := range dirEntries {
		if !e.IsDir() {
			continue
		}

		pid, _, err := FromInstanceDir(filepath.Join(dirPath, e.Name()))
		if err != nil {
			continue
		}

		isAlive, err := ProcRunning(pid)
		switch {
		case err != nil:
			continue
		case isAlive:
			continue
		}

		staleState, err := Load(filepath.Join(dirPath, e.Name()), log)
		if err != nil {
			log.Debug("loading stale state", "name", e.Name(), "err", err.Error())
			continue
		}
		if err = staleState.Finalize(cleaner); err != nil {
			log.Debug("finalizing stale state", "name", e.Name(), "err", err.Error())
		}
	}
	return nil
}
