package container

import (
	"fmt"
	"net/url"
	"os"
)

const (
	FallbackSocketPath = "/var/run/docker.sock"
)

func SocketExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil || stat.IsDir() {
		return false
	}
	return (stat.Mode() & os.ModeSocket) != 0
}

func EnvSocket() (string, error) {
	if host := os.Getenv("DOCKER_HOST"); host != empty {
		dh, err := url.Parse(host)
		if err != nil || dh.Scheme != "unix" {
			return "", fmt.Errorf(
				"getting socket from DOCKER_HOST env var: %s",
				host,
			)
		}
		return dh.Path, nil
	}
	return "", fmt.Errorf("DOCKER_HOST env var not set")
}
