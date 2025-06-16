package state_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/state"
)

func TestInstanceDir(t *testing.T) {
	// given
	wantPID := 123
	wantID := uuid.New()
	wantDir := "test-state-dir"

	// when
	path := state.InstanceDir(wantDir, wantPID, wantID)

	// then
	assert.NotEmpty(t, path)
	assert.Contains(t,
		path,
		filepath.Join(
			wantDir,
			fmt.Sprintf(
				"%d%s%s",
				wantPID,
				state.NameSeparator,
				wantID,
			),
		),
	)
}

func assertFileExistsByName(t *testing.T, dir string, fileName string) {
	elements, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.True(t, slices.ContainsFunc(elements, func(e os.DirEntry) bool {
		return !e.IsDir() && e.Name() == fileName
	}), "Expected to find %s", fileName)
}

func assertDirElementCount(t *testing.T, dir string, expected int) {
	elements, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Equal(
		t,
		expected,
		len(elements),
		"expected to find %d elements in dir: '%s'",
		expected,
		dir,
	)
}

func findDirByNamePrefix(t *testing.T, dir string, prefix string) []string {
	var dirs []string
	elements, err := os.ReadDir(dir)
	assert.NoError(t, err)
	for _, e := range elements {
		if e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			dirs = append(dirs, filepath.Join(dir, e.Name()))
		}
	}
	return dirs
}

func assertCorrectMetadata(t *testing.T, instanceStateDir string) {
	stateFile := filepath.Join(instanceStateDir, "metadata.json")
	stateBytes, err := os.ReadFile(stateFile)
	require.NoError(t, err)

	gotState := struct {
		Time string `json:"time"`
	}{}
	err = json.Unmarshal(stateBytes, &gotState)
	require.NoError(t, err)

	// assert app start time is in the past
	startTime, err := time.Parse(time.RFC3339, gotState.Time)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, time.Now(), startTime)
}

func assertIDsEqual(t *testing.T, instanceStateDir string, wantIds []string) {
	gotIDs, err := state.GetPersistedIDs(instanceStateDir)
	require.NoError(t, err)
	assert.Equal(t, wantIds, gotIDs)
}

func TestState_Init(t *testing.T) {
	t.Run("happy path - one instance", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		appPID := 1

		// when
		got, err := state.Init(appStateDir, appPID)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, got)

		// assert state
		assert.Empty(t, got.IDs())
		assert.True(
			t,
			strings.HasPrefix(
				filepath.Base(got.Dir()),
				fmt.Sprintf(
					"%d%s",
					appPID,
					state.NameSeparator,
				),
			),
			"instance state dir must start with PID",
		)
		assert.GreaterOrEqual(t, time.Now(), got.Time())

		assertDirElementCount(t, appStateDir, 1)

		//assert persisted state
		instanceStateDir := got.Dir()
		assertFileExistsByName(t, instanceStateDir, "metadata.json")
		assertFileExistsByName(t, instanceStateDir, "ids.json")
		assertCorrectMetadata(t, instanceStateDir)
		assertIDsEqual(t, instanceStateDir, []string{})
	})

	t.Run("happy path - two instances", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		PIDs := []int{1, 2}

		// when
		gotInstance1, err := state.Init(appStateDir, PIDs[0])
		require.NoError(t, err)
		gotInstance2, err := state.Init(appStateDir, PIDs[1])
		require.NoError(t, err)

		// then
		assertDirElementCount(t, appStateDir, 2)
		require.NotEmpty(t, gotInstance1)
		require.NotEmpty(t, gotInstance2)
		require.LessOrEqual(t, gotInstance2.Time(), gotInstance2.Time())

		instances := []*state.State{gotInstance1, gotInstance2}
		for idx, instance := range instances {
			// assert state
			assert.Empty(t, instance.IDs())
			assert.True(
				t,
				strings.HasPrefix(
					filepath.Base(instance.Dir()),
					fmt.Sprintf(
						"%d%s",
						PIDs[idx],
						state.NameSeparator,
					),
				),
				"instance state dir must start with PID",
			)
			assert.GreaterOrEqual(t, time.Now(), instance.Time())

			// assert persisted state
			assertFileExistsByName(t, instance.Dir(), "metadata.json")
			assertFileExistsByName(t, instance.Dir(), "ids.json")
			assertCorrectMetadata(t, instance.Dir())
			assertIDsEqual(t, instance.Dir(), []string{})
		}
	})
}

func TestState_Remove(t *testing.T) {
	t.Run("happy path - one instance", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()

		sut, err := state.Init(appStateDir, 1)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		// when
		gotErr := sut.Remove()

		// then
		require.NoError(t, gotErr)
		assertDirElementCount(t, appStateDir, 0)
	})

	t.Run("happy path - two instances (different PIDs)", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()

		sut, err := state.Init(appStateDir, 1)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		_, err = state.Init(appStateDir, 2)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 2)

		// when
		gotErr := sut.Remove()

		// then
		require.NoError(t, gotErr)
		assertDirElementCount(t, appStateDir, 1)
		assert.Len(
			t,
			findDirByNamePrefix(
				t,
				appStateDir,
				fmt.Sprintf("%d%s", 1, state.NameSeparator),
			),
			0,
			"expected no dirs belonging to PID '%d'",
			1,
		)
	})

	t.Run("happy path - two instances (same PIDs)", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()

		sut, err := state.Init(appStateDir, 1)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		_, err = state.Init(appStateDir, 1)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 2)

		// when
		gotErr := sut.Remove()

		// then
		require.NoError(t, gotErr)
		assertDirElementCount(t, appStateDir, 1)
	})
}

func TestState_AddID(t *testing.T) {
	t.Run("happy path - one instance", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		wantIDs := []string{"a", "b", "c"}

		sut, err := state.Init(appStateDir, 1)
		require.NoError(t, err)

		// when
		for _, id := range wantIDs {
			require.NoError(t, sut.AddID(id))
		}

		// then
		assertIDsEqual(t, sut.Dir(), wantIDs)
	})

	for name, tc := range map[string]struct {
		sutPID   int
		otherPID int
	}{
		"happy path - two instances (different PIDs)": {
			sutPID:   1,
			otherPID: 2,
		},
		"happy path - two instances (same PID)": {
			sutPID:   1,
			otherPID: 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			appStateDir := t.TempDir()
			wantIDs := []string{"a", "b", "c"}

			sut, err := state.Init(appStateDir, tc.sutPID)
			require.NoError(t, err)

			otherInstance, err := state.Init(appStateDir, tc.otherPID)
			require.NoError(t, err)

			// when
			for _, id := range wantIDs {
				require.NoError(t, sut.AddID(id))
			}

			// then
			assertIDsEqual(t, sut.Dir(), wantIDs)
			assertIDsEqual(t, otherInstance.Dir(), []string{})
		})
	}
}

func TestState_RemoveID(t *testing.T) {
	t.Run("happy path - one instance", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		IDsToAdd := []string{"a", "b", "c"}
		IDsToRemove := []string{"a", "b"}
		wantIDs := []string{"c"}

		sut, err := state.Init(appStateDir, 1)
		require.NoError(t, err)

		for _, id := range IDsToAdd {
			require.NoError(t, sut.AddID(id))
		}

		// when
		for _, id := range IDsToRemove {
			require.NoError(t, sut.RemoveID(id))
		}

		// then
		assertIDsEqual(t, sut.Dir(), wantIDs)
	})

	for name, tc := range map[string]struct {
		sutPID   int
		otherPID int
	}{
		"happy path - two instances (different PIDs)": {
			sutPID:   1,
			otherPID: 2,
		},
		"happy path - two instances (same PID)": {
			sutPID:   1,
			otherPID: 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			appStateDir := t.TempDir()
			IDsToAdd := []string{"a", "b", "c"}
			IDsToRemove := []string{"a", "b"}
			wantIDs := []string{"c"}

			sut, err := state.Init(appStateDir, tc.sutPID)
			require.NoError(t, err)

			otherInstance, err := state.Init(appStateDir, tc.otherPID)
			require.NoError(t, err)

			for _, id := range IDsToAdd {
				require.NoError(t, sut.AddID(id))
				require.NoError(t, otherInstance.AddID(id))
			}

			// when
			for _, id := range IDsToRemove {
				require.NoError(t, sut.RemoveID(id))
			}

			// then
			assertIDsEqual(t, sut.Dir(), wantIDs)
			assertIDsEqual(t, otherInstance.Dir(), IDsToAdd)
		})
	}
}
