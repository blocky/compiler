package test

import (
	"fmt"
	"os/exec"
)

func ContainerRuntimeAvailable() bool {
	return exec.Command("docker", "version").Run() == nil
}

func RemoveContainersByPrefix(prefix string) *exec.Cmd {
	removeCmd := fmt.Sprintf(`
        cIDs=$(docker ps -a --filter "name=^/%s" -q)
        if [ -n "$cIDs" ]; then
            docker rm -f $cIDs
        fi
    `, prefix)
	return exec.Command("bash", "-c", removeCmd)
}
