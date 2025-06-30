package container_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/container"
)

func TestGetDaemonSocketPath(t *testing.T) {
	t.Run("happy path - socket from env var", func(t *testing.T) {
		// given
		wantSock := "/my/local/docker.sock"
		unixSock := "unix://" + wantSock

		cleanValue := os.Getenv("DOCKER_HOST")
		defer os.Setenv("DOCKER_HOST", cleanValue)
		err := os.Setenv("DOCKER_HOST", unixSock)
		require.NoError(t, err)

		// when
		gotSock := container.GetDaemonSocketPath()

		// then
		assert.Equal(t, wantSock, gotSock)
	})

	t.Run("happy path - socket from fallback value", func(t *testing.T) {
		// when
		gotSock := container.GetDaemonSocketPath()

		// then
		assert.Equal(t, container.FallbackSocketPath, gotSock)
	})
}
