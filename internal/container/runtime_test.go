package container_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/container"
	"github.com/blocky/compiler/mocks"
)

func TestRuntime_Compatible(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		sut := container.NewRuntimeFromRaw(mockClient, nil, slog.Default())

		// expecting
		mockClient.EXPECT().
			Compatible(ctx).
			Return(true, nil).
			Once()

		// when
		got, gotErr := sut.Compatible(ctx)

		// then
		require.NoError(t, gotErr)
		assert.True(t, got)
	})

	t.Run("client error", func(t *testing.T) {
		// given
		wantErrorMsg := "client error"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		sut := container.NewRuntimeFromRaw(mockClient, nil, slog.Default())

		// expecting
		mockClient.EXPECT().
			Compatible(ctx).
			Return(false, errors.New(wantErrorMsg)).
			Once()

		// when
		_, gotErr := sut.Compatible(ctx)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrorMsg)
	})
}

func TestRuntime_GetImage(t *testing.T) {
	t.Run("happy path - image exists", func(t *testing.T) {
		// given
		wantImage := "test:latest"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		sut := container.NewRuntimeFromRaw(mockClient, nil, slog.Default())

		// expecting
		mockClient.EXPECT().
			ImageExists(ctx, wantImage).
			Return(true, nil).
			Once()

		// when
		gotErr := sut.GetImage(ctx, wantImage)

		// then
		require.NoError(t, gotErr)
	})

	t.Run("happy path - no image", func(t *testing.T) {
		// given
		wantImage := "test:latest"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		sut := container.NewRuntimeFromRaw(mockClient, nil, mockLogger)

		// expecting
		mockClient.EXPECT().
			ImageExists(ctx, wantImage).
			Return(false, nil).
			Once()

		mockLogger.EXPECT().
			Debug("Image not available, pulling", "image", wantImage).
			Return().
			Once()

		mockClient.EXPECT().
			PullImage(ctx, wantImage).
			Return(nil).
			Once()

		// when
		gotErr := sut.GetImage(ctx, wantImage)

		// then
		require.NoError(t, gotErr)
	})

	t.Run("error checking if image exists", func(t *testing.T) {
		// given
		wantImage := "test:latest"
		wantErrorMsg := "test error message"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		sut := container.NewRuntimeFromRaw(mockClient, nil, mockLogger)

		// expecting
		mockClient.EXPECT().
			ImageExists(ctx, wantImage).
			Return(false, errors.New(wantErrorMsg)).
			Once()

		// when
		gotErr := sut.GetImage(ctx, wantImage)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrorMsg)
	})

	t.Run("error pulling image", func(t *testing.T) {
		// given
		wantErrorMsg := "test error message"
		wantImage := "test:latest"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		sut := container.NewRuntimeFromRaw(mockClient, nil, mockLogger)

		// expecting
		mockClient.EXPECT().
			ImageExists(ctx, wantImage).
			Return(false, nil).
			Once()

		mockLogger.EXPECT().
			Debug("Image not available, pulling", "image", wantImage).
			Return().
			Once()

		mockClient.EXPECT().
			PullImage(ctx, wantImage).
			Return(errors.New(wantErrorMsg)).
			Once()

		// when
		gotErr := sut.GetImage(ctx, wantImage)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrorMsg)
	})
}

func TestRuntime_Launch(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantCfg := container.Config{}
		wantID := "testid"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(
			mockClient,
			mockMemory,
			slog.Default(),
		)

		// expecting
		mockClient.EXPECT().
			Create(ctx, wantCfg).
			Return(wantID, nil).
			Once()

		mockClient.EXPECT().
			Start(ctx, wantID).
			Return(nil).
			Once()

		mockMemory.EXPECT().
			AddID(wantID).
			Return(nil).
			Once()

		// when
		gotID, gotErr := sut.Launch(ctx, wantCfg)

		// then
		require.NoError(t, gotErr)
		assert.Equal(t, wantID, gotID)
	})

	t.Run("container create error", func(t *testing.T) {
		// given
		wantErrMsg := "container create error"
		wantCfg := container.Config{}
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(
			mockClient,
			mockMemory,
			slog.Default(),
		)

		// expecting
		mockClient.EXPECT().
			Create(ctx, wantCfg).
			Return("", errors.New(wantErrMsg)).
			Once()

		// when
		_, gotErr := sut.Launch(ctx, wantCfg)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrMsg)
	})

	t.Run("container ID save error", func(t *testing.T) {
		// given
		wantErrMsg := "saving container ID to memory"
		wantCfg := container.Config{}
		wantID := "testid"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(
			mockClient,
			mockMemory,
			slog.Default(),
		)

		// expecting
		mockClient.EXPECT().
			Create(ctx, wantCfg).
			Return(wantID, nil).
			Once()

		mockMemory.EXPECT().
			AddID(wantID).
			Return(errors.New(wantErrMsg)).
			Once()

		// when
		_, gotErr := sut.Launch(ctx, wantCfg)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrMsg)
	})

	t.Run("container start error", func(t *testing.T) {
		// given
		wantErrMsg := "container start error"
		wantCfg := container.Config{}
		wantID := "testid"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(
			mockClient,
			mockMemory,
			slog.Default(),
		)

		// expecting
		mockClient.EXPECT().
			Create(ctx, wantCfg).
			Return(wantID, nil).
			Once()

		mockMemory.EXPECT().
			AddID(wantID).
			Return(nil).
			Once()

		mockClient.EXPECT().
			Start(ctx, wantID).
			Return(errors.New(wantErrMsg)).
			Once()

		// when
		_, gotErr := sut.Launch(ctx, wantCfg)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrMsg)
	})
}

func TestRuntime_GetOutput(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantID := "testid"
		wantStatus := 2025
		wantMsg := "test message"
		wantStdOut := "output from stdout"
		wantStdErr := "output from stderr"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		sut := container.NewRuntimeFromRaw(mockClient, nil, slog.Default())

		// expecting
		mockClient.EXPECT().
			Wait(ctx, wantID).
			Return(wantStatus, wantMsg, nil).
			Once()

		mockClient.EXPECT().
			Logs(ctx, wantID).
			Return(wantStdOut, wantStdErr, nil).
			Once()

		// when
		gotOutput, gotErr := sut.GetOutput(ctx, wantID)

		// then
		require.NoError(t, gotErr)
		assert.Equal(t, wantStatus, gotOutput.Status)
		assert.Equal(t, wantMsg, gotOutput.ErrorMsg)
		assert.Equal(t, wantStdOut, gotOutput.StdOut)
		assert.Equal(t, wantStdErr, gotOutput.StdErr)
	})

	t.Run("error waiting for container", func(t *testing.T) {
		// given
		wantID := "testid"
		wantErrorMsg := "test error message"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		sut := container.NewRuntimeFromRaw(mockClient, nil, slog.Default())

		// expecting
		mockClient.EXPECT().
			Wait(ctx, wantID).
			Return(0, "", errors.New(wantErrorMsg)).
			Once()

		// when
		_, gotErr := sut.GetOutput(ctx, wantID)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrorMsg)
	})

	t.Run("error getting container logs", func(t *testing.T) {
		// given
		wantID := "testid"
		wantErrorMsg := "test error message"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		sut := container.NewRuntimeFromRaw(mockClient, nil, slog.Default())

		// expecting
		mockClient.EXPECT().
			Wait(ctx, wantID).
			Return(0, "", nil).
			Once()

		mockClient.EXPECT().
			Logs(ctx, wantID).
			Return("", "", errors.New(wantErrorMsg)).
			Once()

		// when
		_, gotErr := sut.GetOutput(ctx, wantID)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrorMsg)
	})
}

func TestRuntime_CleanUp(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantID := "testid"
		ctx := context.Background()
		mockLogger := mocks.NewContainerLogger(t)
		mockClient := mocks.NewContainerClient(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(
			mockClient,
			mockMemory,
			mockLogger,
		)

		// expecting
		mockClient.EXPECT().
			ContainerExists(ctx, wantID).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			Stop(ctx, wantID).
			Return(nil).
			Once()
		mockClient.EXPECT().
			Remove(ctx, wantID).
			Return(nil).
			Once()

		mockMemory.EXPECT().
			RemoveID(wantID).
			Return(nil).
			Once()

		// when
		gotErr := sut.CleanUp(ctx, wantID)

		// then
		require.NoError(t, gotErr)
	})

	t.Run("container does not exist", func(t *testing.T) {
		// given
		wantID := "testid"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockMemory := mocks.NewContainerMemory(t)
		mockLogger := mocks.NewContainerLogger(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			ContainerExists(ctx, wantID).
			Return(false, nil).
			Once()

		mockLogger.EXPECT().
			Debug("Container doesn't exist", "id", wantID).
			Once()

		// when
		gotErr := sut.CleanUp(ctx, wantID)

		// then
		require.NoError(t, gotErr)
	})

	t.Run("error checking if container exists", func(t *testing.T) {
		// given
		wantID := "testid"
		wantErr := fmt.Errorf("test error message")
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockMemory := mocks.NewContainerMemory(t)
		mockLogger := mocks.NewContainerLogger(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			ContainerExists(ctx, wantID).
			Return(false, wantErr).
			Once()

		// when
		gotErr := sut.CleanUp(ctx, wantID)

		// then
		require.Error(t, gotErr)
		require.ErrorContains(t, gotErr, "checking container status")
	})

	t.Run("error stopping container", func(t *testing.T) {
		// given
		wantID := "testid"
		wantErr := fmt.Errorf("test error message")
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			ContainerExists(ctx, wantID).
			Return(true, nil).
			Once()

		mockClient.EXPECT().
			Stop(ctx, wantID).
			Return(wantErr).
			Once()

		mockLogger.EXPECT().
			Debug("stopping container", "err", wantErr.Error()).
			Once()

		mockClient.EXPECT().
			Remove(ctx, wantID).
			Return(nil).
			Once()

		mockMemory.EXPECT().
			RemoveID(wantID).
			Return(nil).
			Once()

		// when
		gotErr := sut.CleanUp(ctx, wantID)

		// then
		require.NoError(t, gotErr)
	})

	t.Run("error removing container", func(t *testing.T) {
		// given
		wantID := "testid"
		wantErrMsg := "test error message"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, slog.Default())

		// expecting
		mockClient.EXPECT().
			ContainerExists(ctx, wantID).
			Return(true, nil).
			Once()

		mockClient.EXPECT().
			Stop(ctx, wantID).
			Return(nil).
			Once()

		mockClient.EXPECT().
			Remove(ctx, wantID).
			Return(errors.New(wantErrMsg)).
			Once()

		// when
		gotErr := sut.CleanUp(ctx, wantID)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrMsg)
	})

	t.Run("error removing container id from memory", func(t *testing.T) {
		// given
		wantID := "testid"
		wantErr := fmt.Errorf("test error message")
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			ContainerExists(ctx, wantID).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			Stop(ctx, wantID).
			Return(nil).
			Once()

		mockClient.EXPECT().
			Remove(ctx, wantID).
			Return(nil).
			Once()

		mockMemory.EXPECT().
			RemoveID(wantID).
			Return(wantErr).
			Once()

		// when
		gotErr := sut.CleanUp(ctx, wantID)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErr.Error())
	})
}

func TestRuntime_Run(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantImageID := "test:test"
		wantID := "testid"
		givenCfg := container.Config{
			Image: wantImageID,
		}
		wantStatus := container.OK
		wantMsg := "test message"
		wantStdOut := "output from stdout"
		wantStdErr := "output from stderr"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			Compatible(ctx).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			ImageExists(ctx, wantImageID).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			Create(ctx, givenCfg).
			Return(wantID, nil).
			Once()
		mockClient.EXPECT().
			Start(ctx, wantID).
			Return(nil).
			Once()
		mockClient.EXPECT().
			Wait(ctx, wantID).
			Return(wantStatus, wantMsg, nil).
			Once()
		mockClient.EXPECT().
			Logs(ctx, wantID).
			Return(wantStdOut, wantStdErr, nil).
			Once()
		mockClient.EXPECT().
			ContainerExists(ctx, wantID).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			Stop(ctx, wantID).
			Return(nil).
			Once()
		mockClient.EXPECT().
			Remove(ctx, wantID).
			Return(nil).
			Once()

		mockMemory.EXPECT().
			AddID(wantID).
			Return(nil).
			Once()
		mockMemory.EXPECT().
			RemoveID(wantID).
			Return(nil).
			Once()

		mockLogger.EXPECT().
			Debug(wantStdOut).
			Return().
			Once()
		mockLogger.EXPECT().
			Debug(wantStdErr).
			Return().
			Once()

		// when
		gotOutput, gotErr := sut.Run(ctx, givenCfg)

		// then
		require.NoError(t, gotErr)
		assert.Equal(t, wantStatus, gotOutput.Status)
		assert.Equal(t, wantMsg, gotOutput.ErrorMsg)
		assert.Equal(t, wantStdOut, gotOutput.StdOut)
		assert.Equal(t, wantStdErr, gotOutput.StdErr)
	})

	t.Run("runtime not compatible", func(t *testing.T) {
		// given
		wantImageID := "test:test"
		givenCfg := container.Config{
			Image: wantImageID,
		}
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			Compatible(ctx).
			Return(false, nil).
			Once()

		// when
		_, gotErr := sut.Run(ctx, givenCfg)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, "runtime not compatible")
	})

	t.Run("error checking compatibility", func(t *testing.T) {
		// given
		wantImageID := "test:test"
		givenCfg := container.Config{
			Image: wantImageID,
		}
		wantErrMsg := "test error message"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			Compatible(ctx).
			Return(false, errors.New(wantErrMsg)).
			Once()

		// when
		_, gotErr := sut.Run(ctx, givenCfg)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrMsg)
	})

	t.Run("error getting image", func(t *testing.T) {
		// given
		wantImageID := "test:test"
		givenCfg := container.Config{
			Image: wantImageID,
		}
		wantErrMsg := "test error message"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			Compatible(ctx).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			ImageExists(ctx, wantImageID).
			Return(false, errors.New(wantErrMsg)).
			Once()

		// when
		_, gotErr := sut.Run(ctx, givenCfg)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrMsg)
	})

	t.Run("error starting container", func(t *testing.T) {
		// given
		wantImageID := "test:test"
		wantID := "testid"
		givenCfg := container.Config{
			Image: wantImageID,
		}
		wantErrMsg := "test error message"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			Compatible(ctx).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			ImageExists(ctx, wantImageID).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			Create(ctx, givenCfg).
			Return(wantID, nil).
			Once()
		mockClient.EXPECT().
			Start(ctx, wantID).
			Return(errors.New(wantErrMsg)).
			Once()

		mockMemory.EXPECT().
			AddID(wantID).
			Return(nil).
			Once()

		// when
		_, gotErr := sut.Run(ctx, givenCfg)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrMsg)
	})

	t.Run("error getting container output", func(t *testing.T) {
		// given
		wantImageID := "test:test"
		wantID := "testid"
		givenCfg := container.Config{
			Image: wantImageID,
		}
		wantErrMsg := "test error message"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			Compatible(ctx).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			ImageExists(ctx, wantImageID).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			Create(ctx, givenCfg).
			Return(wantID, nil).
			Once()
		mockClient.EXPECT().
			Start(ctx, wantID).
			Return(nil).
			Once()
		mockClient.EXPECT().
			Wait(ctx, wantID).
			Return(1, "", errors.New(wantErrMsg)).
			Once()
		mockClient.EXPECT().
			ContainerExists(ctx, wantID).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			Stop(ctx, wantID).
			Return(nil).
			Once()
		mockClient.EXPECT().
			Remove(ctx, wantID).
			Return(nil).
			Once()

		mockMemory.EXPECT().
			AddID(wantID).
			Return(nil).
			Once()
		mockMemory.EXPECT().
			RemoveID(wantID).
			Return(nil).
			Once()

		// when
		_, gotErr := sut.Run(ctx, givenCfg)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantErrMsg)
	})

	t.Run("error cleaning up container", func(t *testing.T) {
		// given
		wantImageID := "test:test"
		wantID := "testid"
		givenCfg := container.Config{
			Image: wantImageID,
		}
		wantStatus := container.OK
		wantMsg := "test message"
		wantStdOut := "output from stdout"
		wantStdErr := "output from stderr"
		wantErrMsg := "removing container: test error message"
		ctx := context.Background()
		mockClient := mocks.NewContainerClient(t)
		mockLogger := mocks.NewContainerLogger(t)
		mockMemory := mocks.NewContainerMemory(t)
		sut := container.NewRuntimeFromRaw(mockClient, mockMemory, mockLogger)

		// expecting
		mockClient.EXPECT().
			Compatible(ctx).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			ImageExists(ctx, wantImageID).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			Create(ctx, givenCfg).
			Return(wantID, nil).
			Once()
		mockClient.EXPECT().
			Start(ctx, wantID).
			Return(nil).
			Once()
		mockClient.EXPECT().
			Wait(ctx, wantID).
			Return(wantStatus, wantMsg, nil).
			Once()
		mockClient.EXPECT().
			Logs(ctx, wantID).
			Return(wantStdOut, wantStdErr, nil).
			Once()
		mockClient.EXPECT().
			ContainerExists(ctx, wantID).
			Return(true, nil).
			Once()
		mockClient.EXPECT().
			Stop(ctx, wantID).
			Return(nil).
			Once()
		mockClient.EXPECT().
			Remove(ctx, wantID).
			Return(errors.New(wantErrMsg)).
			Once()

		mockLogger.EXPECT().
			Debug(wantStdOut).
			Return().
			Once()
		mockLogger.EXPECT().
			Debug(wantStdErr).
			Return().
			Once()
		mockLogger.EXPECT().
			Error("cleaning up container", "err", mock.Anything).
			Once()

		mockMemory.EXPECT().
			AddID(wantID).
			Return(nil).
			Once()

		// when
		gotOutput, gotErr := sut.Run(ctx, givenCfg)

		// then
		require.NoError(t, gotErr)
		assert.Equal(t, wantStatus, gotOutput.Status)
		assert.Equal(t, wantMsg, gotOutput.ErrorMsg)
		assert.Equal(t, wantStdOut, gotOutput.StdOut)
		assert.Equal(t, wantStdErr, gotOutput.StdErr)
	})
}
