package state_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/state"
	"github.com/blocky/compiler/mocks"
)

const NotPID = -1

func TestCleanupStale(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()

		mockLogger := mocks.NewStateLogger(t)

		_, err := state.Init(appStateDir, validPID)
		require.NoError(t, err)
		staleInstance, err := state.Init(appStateDir, NotPID)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 2)

		wantIDs := []string{"a", "b", "c"}
		mockCleaner := mocks.NewStateCleaner(t)
		for _, ID := range wantIDs {
			err = staleInstance.AddID(ID)
			require.NoError(t, err)

			// expecting
			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(nil).
				Once()
		}

		// when
		err = state.CleanupStale(appStateDir, mockCleaner, mockLogger)

		// then
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)
		assert.Len(
			t,
			findDirByNamePrefix(
				t,
				appStateDir,
				fmt.Sprintf("%d%s", NotPID, state.NameSeparator),
			),
			0,
			"expected no dirs belonging to PID '%d'",
			1,
		)
		assert.Len(
			t,
			findDirByNamePrefix(
				t,
				appStateDir,
				fmt.Sprintf("%d%s", validPID, state.NameSeparator),
			),
			1,
			"expected no dirs belonging to PID '%d'",
			1,
		)
	})

	t.Run("happy path - ignore unknown dir elements", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()

		_, err := state.Init(appStateDir, validPID)
		require.NoError(t, err)
		_, err = state.Init(appStateDir, NotPID)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 2)

		files := []string{
			"aFile.txt",
		}
		for _, file := range files {
			require.NoError(
				t,
				os.WriteFile(
					filepath.Join(appStateDir, file),
					[]byte{},
					0600,
				),
			)
		}

		dirs := []string{
			"dir-with-no-pid-separator",
			"dir.with.too.many.pid.separators",
			"abc.unparsable-pid",
		}
		for _, dir := range dirs {
			require.NoError(
				t,
				os.Mkdir(
					filepath.Join(appStateDir, dir),
					0700,
				),
			)
		}

		// when
		err = state.CleanupStale(appStateDir, nil, nil)
		require.NoError(t, err)

		// then
		assertDirElementCount(t, appStateDir, len(files)+len(dirs)+2-1)
		assert.Len(
			t,
			findDirByNamePrefix(
				t,
				appStateDir,
				fmt.Sprintf("%d%s", NotPID, state.NameSeparator),
			),
			0,
			"expected no dirs belonging to PID '%d'",
			1,
		)
		assert.Len(
			t,
			findDirByNamePrefix(
				t,
				appStateDir,
				fmt.Sprintf("%d%s", validPID, state.NameSeparator),
			),
			1,
			"expected no dirs belonging to PID '%d'",
			1,
		)
	})

	t.Run("error reading IDs", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()

		staleInstance, err := state.Init(appStateDir, NotPID)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)
		err = os.Remove(filepath.Join(staleInstance.Dir(), "ids.json"))
		require.NoError(t, err)

		// expecting
		mockLogger := mocks.NewStateLogger(t)
		mockLogger.EXPECT().
			Debug("error getting persisted ids", "name", mock.Anything).
			Once()

		// when
		err = state.CleanupStale(appStateDir, nil, mockLogger)

		// then
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)
		assert.Len(
			t,
			findDirByNamePrefix(
				t,
				appStateDir,
				fmt.Sprintf("%d%s", NotPID, state.NameSeparator),
			),
			1,
			"expected no dirs belonging to PID '%d'",
			1,
		)

	})

	t.Run("error cleaning an ID", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		mockLogger := mocks.NewStateLogger(t)

		staleInstance, err := state.Init(appStateDir, NotPID)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		wantIDs := []string{"a", "b", "c"}
		mockCleaner := mocks.NewStateCleaner(t)
		wantError := errors.New("cleanup error")
		for _, ID := range wantIDs {
			err = staleInstance.AddID(ID)
			require.NoError(t, err)

			// expecting
			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(wantError).
				Once()
			mockLogger.EXPECT().
				Debug("cleaning up id", "id", ID, "err", wantError.Error()).
				Once()
		}

		// when
		err = state.CleanupStale(appStateDir, mockCleaner, mockLogger)

		// then
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 0)
	})

	t.Run("error reading state directory", func(t *testing.T) {
		// given
		appStateDir := "/dir-that-does-not-exist"

		// when
		err := state.CleanupStale(appStateDir, nil, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "reading app state dir")
	})

	t.Run("error removing stale state directory", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()

		staleInstance, err := state.Init(appStateDir, NotPID)
		require.NoError(t, err)
		err = os.Chmod(staleInstance.Dir(), 0555)
		require.NoError(t, err)
		defer func() {
			_ = os.Chmod(staleInstance.Dir(), 0700)
		}()
		assertDirElementCount(t, appStateDir, 1)

		// when
		err = state.CleanupStale(appStateDir, nil, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "removing stale state directory")
		assertDirElementCount(t, appStateDir, 1)
	})
}

func TestCleanIDs(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		mockCleaner := mocks.NewStateCleaner(t)

		// expecting
		wantIDs := []string{"a", "b", "c"}
		for _, ID := range wantIDs {
			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(nil).
				Once()
		}

		// when
		state.CleanIDs(mockCleaner, nil, wantIDs)
	})

	t.Run("error cleaning up id", func(t *testing.T) {
		// given
		mockCleaner := mocks.NewStateCleaner(t)
		mockLogger := mocks.NewStateLogger(t)

		// expecting
		wantError := errors.New("test cleanup error")
		wantIDs := []string{"a", "b", "c"}
		for _, ID := range wantIDs {
			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(wantError).
				Once()
			mockLogger.EXPECT().
				Debug("cleaning up id", "id", ID, "err", wantError.Error()).
				Once()
		}

		// when
		state.CleanIDs(mockCleaner, mockLogger, wantIDs)
	})
}

func TestGetPersistedIDs(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		dir := t.TempDir()

		wantIDs := []string{"a", "b", "c"}
		wantBytes, err := json.Marshal(wantIDs)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(dir, "ids.json"), wantBytes, 0600)
		require.NoError(t, err)

		// when
		gotIDs, err := state.GetPersistedIDs(dir)

		// then
		require.NoError(t, err)
		assert.Equal(t, wantIDs, gotIDs)
	})

	t.Run("error reading ids", func(t *testing.T) {
		// given
		dir := t.TempDir()
		// when
		_, err := state.GetPersistedIDs(dir)

		// then
		require.Error(t, err)
		require.ErrorContains(t, err, "reading ids")
	})

	t.Run("error unmarshalling ids", func(t *testing.T) {
		// given
		dir := t.TempDir()

		err := os.WriteFile(filepath.Join(dir, "ids.json"), []byte("wrong-id-file"), 0600)
		require.NoError(t, err)

		// when
		_, err = state.GetPersistedIDs(dir)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unmarshalling ids")
	})
}

func TestProcRunning(t *testing.T) {
	t.Run("happy path - process exists", func(t *testing.T) {
		// given
		wantPID := os.Getpid()

		// when
		ok, err := state.ProcRunning(wantPID)

		// then
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("happy path - process does not exits", func(t *testing.T) {
		// given
		wantPID := -1

		// when
		ok, err := state.ProcRunning(wantPID)

		// then
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
