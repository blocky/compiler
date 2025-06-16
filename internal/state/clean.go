package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type Cleaner interface {
	CleanUp(context.Context, string) error
}

type Logger interface {
	Debug(string, ...any)
}

func procRunning(pid int) (bool, error) {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("finding proc '%d: %w'", pid, err)
	}
	return p.Signal(syscall.Signal(0)) == nil, nil
}

func GetPersistedIDs(dirPath string) ([]string, error) {
	idFile := filepath.Join(dirPath, "ids.json")
	idBytes, err := os.ReadFile(idFile)
	if err != nil {
		return nil, fmt.Errorf("reading ids: %w", err)
	}

	var gotIDs []string
	err = json.Unmarshal(idBytes, &gotIDs)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling ids: %w", err)
	}
	return gotIDs, nil
}

func CleanIDs(cleaner Cleaner, log Logger, IDs []string) {
	ctx := context.Background()
	for _, ID := range IDs {
		err := cleaner.CleanUp(ctx, ID)
		if err != nil {
			log.Debug("cleaning up id", "id", ID)
			continue
		}
	}
}

func CleanupStale(cleaner Cleaner, log Logger, dirPath string) error {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("reading app state dir: %w", err)
	}
	for _, e := range dirEntries {
		if !e.IsDir() {
			continue
		}

		parts := strings.Split(e.Name(), NameSeparator)
		if len(parts) != 2 {
			continue
		}

		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		isAlive, err := procRunning(pid)
		switch {
		case err != nil:
			continue
		case isAlive:
			continue
		}

		staleIDs, err := GetPersistedIDs(filepath.Join(dirPath, e.Name()))
		if err != nil {
			log.Debug("getting persisted ids", "id", e.Name())
			continue
		}
		CleanIDs(cleaner, log, staleIDs)

		if err := os.RemoveAll(filepath.Join(dirPath, e.Name())); err != nil {
			return fmt.Errorf("removing stale state directory: %w", err)
		}
	}
	return nil
}
