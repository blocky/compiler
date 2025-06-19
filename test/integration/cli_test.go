package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/blocky/compiler/internal/bkyc"
)

const (
	srcCodeDir = "testdata"
	scriptDir  = "scripts"
	cliName    = "bky-c"
	tinyGoUID  = 1000
	tinyGoGID  = 1000
	rootEUID   = 0
)

func CLIPath() string {
	return filepath.Join("..", "..", "cmd", "bky-c", "main.go")
}

func runsAsRoot() bool {
	return os.Geteuid() == rootEUID
}

func TestBuild(t *testing.T) {
	t.Cleanup(func() {
		removeContainersByPrefix(bkyc.BkycPrefix)
	})

	projectToBuild := "hello-world-hash-unvendored-go"
	projectDir := filepath.Join(srcCodeDir, projectToBuild)
	envVars := []string{
		"DOCKER_HOST",
	}

	NewProjectTest(t, projectDir).
		BuildIfMissing(CLIPath(), cliName).
		ImportEnvVars(envVars, false).
		CopyDir("in").
		CopyDir("want").
		MakeDir("got").
		ChownIf(runsAsRoot, "got", tinyGoUID, tinyGoGID).
		RunScript(filepath.Join(scriptDir, projectToBuild+".txtar"))
}
