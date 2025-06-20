package state_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/state"
	"github.com/blocky/compiler/mocks"
)

const StalePID = -1

func TestCleanupStale(t *testing.T) {
	t.Run("happy path - alive state", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()

		mockLogger := mocks.NewStateLogger(t)
		mockCleaner := mocks.NewStateCleaner(t)

		_, err := state.Init(appStateDir, validPID, mockLogger)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

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
				fmt.Sprintf("%d%s", validPID, state.NameSeparator),
			),
			1,
			"expected 1 dir belonging to PID '%d'",
			validPID,
		)
	})

	t.Run("happy path - stale state", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		mockLogger := mocks.NewStateLogger(t)

		staleInstance, err := state.Init(appStateDir, StalePID, mockLogger)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

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
		assertDirElementCount(t, appStateDir, 0)
	})

	t.Run("happy path - ignore unknown dir elements", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()

		_, err := state.Init(appStateDir, validPID, slog.Default())
		require.NoError(t, err)
		_, err = state.Init(appStateDir, StalePID, slog.Default())
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
		err = state.CleanupStale(appStateDir, nil, slog.Default())
		require.NoError(t, err)

		// then
		assertDirElementCount(t, appStateDir, len(files)+len(dirs)+1)
		assert.Len(
			t,
			findDirByNamePrefix(
				t,
				appStateDir,
				fmt.Sprintf("%d%s", StalePID, state.NameSeparator),
			),
			0,
			"expected no dirs belonging to PID '%d'",
			StalePID,
		)
		assert.Len(
			t,
			findDirByNamePrefix(
				t,
				appStateDir,
				fmt.Sprintf("%d%s", validPID, state.NameSeparator),
			),
			1,
			"expected %d dirs belonging to PID '%d'",
			1,
			validPID,
		)
	})

	t.Run("error loading stale state", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		mockLogger := mocks.NewStateLogger(t)

		staleInstance, err := state.Init(appStateDir, StalePID, mockLogger)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)
		err = os.Remove(filepath.Join(staleInstance.Dir(), "ids.json"))
		require.NoError(t, err)

		// expecting

		mockLogger.EXPECT().
			Debug(
				"loading stale state",
				"name",
				mock.Anything,
				"err",
				mock.Anything,
			).
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
				fmt.Sprintf("%d%s", StalePID, state.NameSeparator),
			),
			1,
			"expected %d dirs belonging to PID '%d'",
			1,
			StalePID,
		)

	})

	t.Run("error cleaning some IDs", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		mockLogger := mocks.NewStateLogger(t)

		staleInstance, err := state.Init(appStateDir, StalePID, mockLogger)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		wantIDs := []string{"a", "b", "c"}
		mockCleaner := mocks.NewStateCleaner(t)
		wantError := errors.New("cleanup error")
		for _, ID := range wantIDs {
			err = staleInstance.AddID(ID)
			require.NoError(t, err)
		}
		wantErroringIDs := wantIDs[:len(wantIDs)-1]
		for _, ID := range wantErroringIDs {
			// expecting
			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(wantError).
				Once()
			mockLogger.EXPECT().
				Debug("cleaning up id", "id", ID, "err", wantError.Error()).
				Once()
		}
		mockCleaner.EXPECT().
			CleanUp(context.Background(), wantIDs[len(wantIDs)-1]).
			Return(nil).
			Once()

		// when
		err = state.CleanupStale(appStateDir, mockCleaner, mockLogger)

		// then
		require.NoError(t, err)
		assertIDsEqual(t, staleInstance.Dir(), wantErroringIDs)
		assertDirElementCount(t, appStateDir, 1)
	})

	t.Run("error reading state directory", func(t *testing.T) {
		// given
		appStateDir := "/dir-that-does-not-exist"

		// when
		err := state.CleanupStale(appStateDir, nil, slog.Default())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "reading app state dir")
	})

	t.Run("error finalizing stale state", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		mockLogger := mocks.NewStateLogger(t)

		staleInstance, err := state.Init(appStateDir, StalePID, mockLogger)
		require.NoError(t, err)
		err = os.Chmod(staleInstance.Dir(), 0555)
		require.NoError(t, err)
		defer func() {
			_ = os.Chmod(staleInstance.Dir(), 0700)
		}()
		assertDirElementCount(t, appStateDir, 1)

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

		mockLogger.EXPECT().
			Debug(
				"finalizing stale state",
				"name",
				mock.Anything,
				"err",
				mock.Anything,
			).
			Once()

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
				fmt.Sprintf("%d%s", StalePID, state.NameSeparator),
			),
			1,
			"expected %s dir belonging to PID '%d'",
			1,
			StalePID,
		)
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

	t.Run("happy path - process does not exist", func(t *testing.T) {
		// given
		wantPID := -1

		// when
		ok, err := state.ProcRunning(wantPID)

		// then
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
