package container

func GetDaemonSocketPath() string {
	if envSock, err := EnvSocket(); err == nil {
		return envSock
	}
	return FallbackSocketPath
}
