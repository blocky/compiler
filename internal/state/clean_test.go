package state_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
		err = state.CleanupStale(mockCleaner, mockLogger, appStateDir)

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

		//dirsToGoThru := []string{
		//	fmt.Sprintf("aFile.txt"),
		//}

		// when
		err = state.CleanupStale(nil, nil, appStateDir)
		require.NoError(t, err)

		// then
		assertDirElementCount(t, appStateDir, 1)
	})

	t.Run("reading IDs", func(t *testing.T) {

	})

	t.Run("cleaning an ID", func(t *testing.T) {

	})

	t.Run("reading state directory", func(t *testing.T) {

	})

	t.Run("removing stale state directory", func(t *testing.T) {

	})
}

func TestCleanIDs(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {

	})
}
