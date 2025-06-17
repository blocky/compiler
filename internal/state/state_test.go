package state_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/state"
	"github.com/blocky/compiler/mocks"
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

func getPersistedIDs(dirPath string) ([]string, error) {
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

func assertIDsEqual(t *testing.T, instanceStateDir string, wantIds []string) {
	gotIDs, err := getPersistedIDs(instanceStateDir)
	require.NoError(t, err)
	assert.Equal(t, wantIds, gotIDs)
}

func TestInit(t *testing.T) {
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

	for name, tc := range map[string]struct {
		firstPID  int
		secondPID int
	}{
		"happy path - two instances (different PIDs)": {
			firstPID:  1,
			secondPID: 2,
		},
		"happy path - two instances (same PID)": {
			firstPID:  1,
			secondPID: 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			appStateDir := t.TempDir()
			PIDs := []int{tc.firstPID, tc.secondPID}

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
			require.NotEqual(t, gotInstance2.Dir(), gotInstance1.Dir())

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

	t.Run("error saving state", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		appPID := 1
		err := os.Chmod(appStateDir, 0555)
		require.NoError(t, err)
		defer func() {
			_ = os.Chmod(appStateDir, 0700)
		}()

		// when
		_, err = state.Init(appStateDir, appPID)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "saving new state")
	})
}

func TestLoad(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		appPID := 1

		initializedInstance, err := state.Init(appStateDir, appPID)
		require.NoError(t, err)

		wantIDs := []string{"a", "b", "b"}
		for _, ID := range wantIDs {
			err := initializedInstance.AddID(ID)
			require.NoError(t, err)
		}

		// when
		loadedInstance, err := state.Load(initializedInstance.Dir())

		// then
		require.NoError(t, err)
		assert.Equal(t, initializedInstance.IDs(), loadedInstance.IDs())
		assert.Equal(
			t,
			initializedInstance.Time().Format(time.RFC3339),
			loadedInstance.Time().Format(time.RFC3339),
		)
		assert.Equal(t, initializedInstance.Dir(), loadedInstance.Dir())

		// assert state
		assert.True(
			t,
			strings.HasPrefix(
				filepath.Base(loadedInstance.Dir()),
				fmt.Sprintf(
					"%d%s",
					appPID,
					state.NameSeparator,
				),
			),
			"instance state dir must start with PID",
		)
		assert.GreaterOrEqual(t, time.Now(), loadedInstance.Time())
		assertDirElementCount(t, appStateDir, 1)

		//assert persisted state
		instanceStateDir := loadedInstance.Dir()
		assertFileExistsByName(t, instanceStateDir, "metadata.json")
		assertFileExistsByName(t, instanceStateDir, "ids.json")
		assertCorrectMetadata(t, instanceStateDir)
	})

	t.Run("error loading metadata", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		appPID := 1

		initializedInstance, err := state.Init(appStateDir, appPID)
		require.NoError(t, err)

		err = os.Remove(filepath.Join(initializedInstance.Dir(), "metadata.json"))
		require.NoError(t, err)

		// when
		_, err = state.Load(initializedInstance.Dir())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "loading metadata")
	})

	t.Run("error loading ids", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		appPID := 1

		initializedInstance, err := state.Init(appStateDir, appPID)
		require.NoError(t, err)

		err = os.Remove(filepath.Join(initializedInstance.Dir(), "ids.json"))
		require.NoError(t, err)

		// when
		_, err = state.Load(initializedInstance.Dir())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "loading ids")
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
		assert.Equal(t, wantIDs, sut.IDs())
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
			assert.Equal(t, wantIDs, sut.IDs())
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
		assert.Equal(t, wantIDs, sut.IDs())
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
			assert.Equal(t, wantIDs, sut.IDs())
			assertIDsEqual(t, sut.Dir(), wantIDs)
			assertIDsEqual(t, otherInstance.Dir(), IDsToAdd)
		})
	}
}

func TestState_CleanIDs(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()

		sut, err := state.Init(appStateDir, validPID)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		wantIDs := []string{"a", "b", "c"}
		mockCleaner := mocks.NewStateCleaner(t)
		for _, ID := range wantIDs {
			err = sut.AddID(ID)
			require.NoError(t, err)

			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(nil).
				Once()
		}

		// when
		sut.CleanIDs(mockCleaner, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{}, sut.IDs())
		assertIDsEqual(t, sut.Dir(), []string{})
		assertDirElementCount(t, appStateDir, 1)
	})

	t.Run("error cleaning some IDs", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()
		mockLogger := mocks.NewStateLogger(t)

		sut, err := state.Init(appStateDir, validPID)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		wantIDs := []string{"a", "b", "c"}
		mockCleaner := mocks.NewStateCleaner(t)
		wantError := errors.New("cleanup error")
		for _, ID := range wantIDs {
			err = sut.AddID(ID)
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
		sut.CleanIDs(mockCleaner, mockLogger)

		// then
		assert.Equal(t, wantErroringIDs, sut.IDs())
		assertIDsEqual(t, sut.Dir(), wantErroringIDs)
		assertDirElementCount(t, appStateDir, 1)
	})
}

func TestState_Finalize(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()

		sut, err := state.Init(appStateDir, validPID)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		wantIDs := []string{"a", "b", "c"}
		mockCleaner := mocks.NewStateCleaner(t)
		for _, ID := range wantIDs {
			err = sut.AddID(ID)
			require.NoError(t, err)

			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(nil).
				Once()
		}

		// when
		err = sut.Finalize(mockCleaner, nil)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{}, sut.IDs())
		assertDirElementCount(t, appStateDir, 0)
	})

	t.Run("error cleaning some IDs", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()
		mockLogger := mocks.NewStateLogger(t)

		sut, err := state.Init(appStateDir, validPID)
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)

		wantIDs := []string{"a", "b", "c"}
		mockCleaner := mocks.NewStateCleaner(t)
		wantError := errors.New("cleanup error")
		for _, ID := range wantIDs {
			err = sut.AddID(ID)
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
		err = sut.Finalize(mockCleaner, mockLogger)

		// then
		require.NoError(t, err)
		assert.Equal(t, wantErroringIDs, sut.IDs())
		assertIDsEqual(t, sut.Dir(), wantErroringIDs)
		assertDirElementCount(t, appStateDir, 1)
	})

	t.Run("error finalizing stale state", func(t *testing.T) {
		// given
		appStateDir := t.TempDir()
		validPID := os.Getpid()

		sut, err := state.Init(appStateDir, validPID)
		require.NoError(t, err)
		err = os.Chmod(sut.Dir(), 0555)
		require.NoError(t, err)
		defer func() {
			_ = os.Chmod(sut.Dir(), 0700)
		}()
		assertDirElementCount(t, appStateDir, 1)

		wantIDs := []string{"a", "b", "c"}
		mockCleaner := mocks.NewStateCleaner(t)
		for _, ID := range wantIDs {
			err = sut.AddID(ID)
			require.NoError(t, err)

			// expecting
			mockCleaner.EXPECT().
				CleanUp(context.Background(), ID).
				Return(nil).
				Once()
		}
		mockLogger := mocks.NewStateLogger(t)
		mockLogger.EXPECT().
			Debug("removing finalized state", "err", mock.Anything).
			Once()

		// when
		err = sut.Finalize(mockCleaner, mockLogger)

		// then
		require.NoError(t, err)
		assertDirElementCount(t, appStateDir, 1)
	})
}
