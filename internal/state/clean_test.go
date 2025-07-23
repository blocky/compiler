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
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/logsniffer"
	"github.com/blocky/compiler/internal/state"
	"github.com/blocky/compiler/mocks"
)

const StalePID = -1

func TestCleanupStale(t *testing.T) {
	t.Run("happy path - alive state", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()

		mockCleaner := mocks.NewStateCleaner(t)
		defLogger := slog.Default()

		_, err := state.Init(appStateDir, validPID, defLogger)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		// when
		err = state.CleanupStale(appStateDir, mockCleaner, defLogger)

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

		wantIDs := []string{"a", "b", "c"}
		prepareStaleStateDir(t, appStateDir, wantIDs)
		assertDirElementCount(t, appStateDir, 1)

		mockCleaner := mocks.NewStateCleaner(t)
		for _, ID := range wantIDs {
			// expecting
			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(nil).
				Once()
		}

		// when
		err := state.CleanupStale(appStateDir, mockCleaner, slog.Default())

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
		prepareStaleStateDir(t, appStateDir, []string{})
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
		testLogger := logsniffer.NewLogger()

		staleInstance, err := state.Init(appStateDir, StalePID, testLogger.Slog())
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)
		err = os.Remove(filepath.Join(staleInstance.Dir(), "ids.json"))
		require.NoError(t, err)

		// when
		err = state.CleanupStale(appStateDir, nil, testLogger.Slog())

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
		assert.Equal(
			t,
			1,
			testLogger.RecordCount(
				slog.LevelDebug,
				"loading stale state",
			),
		)
	})

	t.Run("error cleaning some IDs", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		wantIDs := []string{"a", "b", "c"}
		dirInfo := prepareStaleStateDir(t, appStateDir, wantIDs)
		assertDirElementCount(t, appStateDir, 1)

		testLogger := logsniffer.NewLogger()

		mockCleaner := mocks.NewStateCleaner(t)
		wantError := errors.New("cleanup error")
		wantErroringIDs := wantIDs[:len(wantIDs)-1]
		for _, ID := range wantErroringIDs {
			// expecting
			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(wantError).
				Once()
		}
		mockCleaner.EXPECT().
			CleanUp(context.Background(), wantIDs[len(wantIDs)-1]).
			Return(nil).
			Once()

		// when
		err := state.CleanupStale(appStateDir, mockCleaner, testLogger.Slog())

		// then
		require.NoError(t, err)
		assertIDsEqual(t, dirInfo.dir, wantErroringIDs)
		assertDirElementCount(t, appStateDir, 1)
		for _, ID := range wantErroringIDs {
			assert.Equal(
				t,
				1,
				testLogger.RecordCountWithAttrs(
					slog.LevelDebug,
					"cleaning up id",
					[]slog.Attr{
						slog.String("id", ID),
						slog.String("err", wantError.Error()),
					},
				),
			)
		}
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

		wantIDs := []string{"a", "b", "c"}
		dirInfo := prepareStaleStateDir(t, appStateDir, wantIDs)
		err := os.Chmod(dirInfo.dir, 0555)
		require.NoError(t, err)
		defer func() {
			_ = os.Chmod(dirInfo.dir, 0700)
		}()
		assertDirElementCount(t, appStateDir, 1)

		testLogger := logsniffer.NewLogger()

		mockCleaner := mocks.NewStateCleaner(t)
		for _, ID := range wantIDs {
			// expecting
			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(nil).
				Once()
		}

		// when
		err = state.CleanupStale(appStateDir, mockCleaner, testLogger.Slog())

		// then
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)
		assert.Len(
			t,
			findDirByNamePrefix(
				t,
				appStateDir,
				fmt.Sprintf("%d%s", dirInfo.pid, state.NameSeparator),
			),
			1,
			"expected %s dir belonging to PID '%d'",
			1,
			dirInfo.pid,
		)

		assert.Equal(
			t,
			1,
			testLogger.RecordCount(
				slog.LevelDebug,
				"finalizing stale state",
			),
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
