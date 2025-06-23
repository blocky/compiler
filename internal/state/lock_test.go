package state_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/state"
)

func LockFromADifferentProc(path string) (string, error) {
	pathToRun := filepath.Join(".", "testdata", "lock", "main.go")
	args := fmt.Sprintf("--path=%s", path)
	cmd := exec.Command("go", "run", pathToRun, args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("locking path '%s': %s: %w", path, out, err)
	}
	return string(out), nil
}

func TestLock(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		lock, err := state.Lock(dir)

		// then
		require.NoError(t, err)
		assert.NotNil(t, lock)
	})

	t.Run("happy path - sequence of lock/unlock", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		lock, err := state.Lock(dir)
		require.NoError(t, err)
		err = lock.Close()
		require.NoError(t, err)

		lock, err = state.Lock(dir)

		// then
		require.NoError(t, err)
		require.NoError(t, lock.Close())
	})

	t.Run("lock already acquired from same process", func(t *testing.T) {
		// given
		dir := t.TempDir()
		lock, err := state.Lock(dir)
		require.NoError(t, err)
		defer lock.Close()

		// when
		_, err = state.Lock(dir)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "acquiring lock")
	})

	t.Run("lock already acquired from a different process", func(t *testing.T) {
		// given
		dir := t.TempDir()
		_, err := state.Lock(dir)
		require.NoError(t, err)

		// when
		_, err = LockFromADifferentProc(dir)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "locking path")
		assert.ErrorContains(t, err, "acquiring lock")
	})

	t.Run("no such folder", func(t *testing.T) {
		// given
		noSuchDir := "/no-such/dir-exists"

		// when
		_, err := state.Lock(noSuchDir)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "opening handle")
	})
}
