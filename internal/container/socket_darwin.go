package container

import (
	"fmt"
	"os/user"
	"path/filepath"
)

var CurrentUser = user.Current

func HomeDirSockPath(homeDir string) string {
	return filepath.Join(homeDir, ".docker", "run", "docker.sock")
}

func UserSocket() (string, error) {
	usr, err := CurrentUser()
	if err != nil {
		return "", fmt.Errorf("getting current user: %v", err)
	}
	usrSock := HomeDirSockPath(usr.HomeDir)
	if !SocketExists(usrSock) {
		return "", fmt.Errorf("no user socker found: %s", usrSock)
	}
	return usrSock, nil
}

func GetDaemonSocketPath() string {
	if envSock, err := EnvSocket(); err == nil {
		return envSock
	}
	if usrSock, err := UserSocket(); err == nil {
		return usrSock
	}
	return FallbackSocketPath
}
