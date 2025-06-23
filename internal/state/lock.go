package state

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func Lock(dirPath string) (*os.File, error) {
	f, err := os.Open(dirPath)
	if err != nil {
		return nil, fmt.Errorf("opening handle: %w", err)
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		wErr := f.Close()
		return nil, errors.Join(
			fmt.Errorf("acquiring lock: %w", err),
			wErr,
		)
	}
	return f, nil
}
