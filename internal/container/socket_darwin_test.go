package container_test

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/container"
)

func TestHomeDirSockPath(t *testing.T) {
	// given
	homeDir := t.TempDir()

	// when
	gotPath := container.HomeDirSockPath(homeDir)

	// then
	assert.Equal(
		t,
		filepath.Join(homeDir, ".docker", "run", "docker.sock"),
		gotPath,
	)
}

func TestGetDaemonSocketPathDarwin(t *testing.T) {
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

	t.Run("happy path - socket from user home dir", func(t *testing.T) {
		// given
		wantHomeDir, err := os.MkdirTemp("", "sock_test")
		require.NoError(t, err)
		defer func() {
			err := os.RemoveAll(wantHomeDir)
			require.NoError(t, err)
		}()

		wantSockDir := filepath.Join(wantHomeDir, ".docker", "run")
		err = os.MkdirAll(wantSockDir, 0777)
		require.NoError(t, err)

		wantSockName := "docker.sock"
		openUnixSocket(t, wantSockDir, wantSockName)
		wantSock := filepath.Join(wantSockDir, wantSockName)

		originalCurrentUser := container.CurrentUser
		container.CurrentUser = func() (*user.User, error) {
			return &user.User{HomeDir: wantHomeDir}, nil
		}
		defer func() {
			container.CurrentUser = originalCurrentUser
		}()

		// when
		gotSock := container.GetDaemonSocketPath()

		// then
		assert.Equal(t, wantSock, gotSock)
	})

	t.Run("happy path - socket from fallback value", func(t *testing.T) {
		// given
		cleanValue := os.Getenv("DOCKER_HOST")
		defer func() {
			err := os.Setenv("DOCKER_HOST", cleanValue)
			require.NoError(t, err)
		}()
		err := os.Setenv("DOCKER_HOST", "")
		require.NoError(t, err)
		
		// when
		gotSock := container.GetDaemonSocketPath()

		// then
		assert.Equal(t, container.FallbackSocketPath, gotSock)
	})
}
