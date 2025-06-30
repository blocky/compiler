package container_test

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/container"
)

func openUnixSocket(t *testing.T, dir string, name string) net.Listener {
	sp := filepath.Join(dir, name)
	l, err := net.Listen("unix", sp)
	require.NoError(t, err)
	return l
}

func TestSocketExists(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		dir := t.TempDir()
		name := "my.sock"
		socket := openUnixSocket(t, dir, name)
		defer socket.Close()

		// when
		ok := container.SocketExists(socket.Addr().String())

		// then
		assert.True(t, ok)
	})

	t.Run("path does not exist", func(t *testing.T) {
		// when
		ok := container.SocketExists(
			filepath.Join("path", "that", "does-not-exist.sock"),
		)

		// then
		assert.False(t, ok)
	})

	t.Run("path points to a regular file", func(t *testing.T) {
		// given
		dir := t.TempDir()
		name := "my.sock"
		path := filepath.Join(dir, name)
		require.NoError(
			t,
			os.WriteFile(
				path,
				[]byte("not-a-socket-file"),
				0o777,
			),
		)

		// when
		ok := container.SocketExists(path)

		// then
		assert.False(t, ok)
	})

	t.Run("path points to a dir", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		ok := container.SocketExists(dir)

		// then
		assert.False(t, ok)
	})
}

func TestEnvSocket(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantSock := "/var/run/docker.sock"
		unixSock := "unix://" + wantSock

		cleanValue := os.Getenv("DOCKER_HOST")
		defer os.Setenv("DOCKER_HOST", cleanValue)
		err := os.Setenv("DOCKER_HOST", unixSock)
		require.NoError(t, err)

		// when
		gotSock, err := container.EnvSocket()

		// then
		require.NoError(t, err)
		assert.Equal(t, wantSock, gotSock)
	})

	for name, tc := range map[string]struct {
		envVarValue string
		wantMsg     string
	}{
		"incorrect scheme": {
			envVarValue: "tcp://localhost/var/run/docker.sock",
			wantMsg:     "getting socket from DOCKER_HOST env var",
		},
		"non-url value": {
			envVarValue: "abcdefghijklmnopqrstuvwxyz",
			wantMsg:     "getting socket from DOCKER_HOST env var",
		},
		"empty value": {
			envVarValue: "",
			wantMsg:     "DOCKER_HOST env var not set",
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			cleanValue := os.Getenv("DOCKER_HOST")
			defer os.Setenv("DOCKER_HOST", cleanValue)
			err := os.Setenv("DOCKER_HOST", tc.envVarValue)
			require.NoError(t, err)

			// when
			_, err = container.EnvSocket()

			// then
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantMsg)
		})
	}
}
