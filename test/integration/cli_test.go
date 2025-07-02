package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/blocky/compiler/cmd/bky-c/cmd"
	"github.com/blocky/compiler/internal/bkyc"
)

const (
	srcCodeDir = "testdata"
	scriptDir  = "scripts"
	tinyGoUID  = 1000
	tinyGoGID  = 1000
	rootEUID   = 0
)

func CLIPath() string {
	return filepath.Join("..", "..", "cmd", cmd.CliName, "main.go")
}

func runsAsRoot() bool {
	return os.Geteuid() == rootEUID
}

func TestBuild(t *testing.T) {
	t.Cleanup(func() {
		removeContainersByPrefix(bkyc.BkycPrefix)
	})

	prepareEnv := func(projectDir string) *ProjectTest {
		envVars := []string{
			"DOCKER_HOST",
		}
		return NewProjectTest(t, projectDir).
			BuildIfMissing(CLIPath(), cmd.CliName).
			MakeDir("xdgHome").
			SetXdgStateHomeDir("xdgHome").
			MakeDir("got").
			ChownIf(runsAsRoot, "got", bkyc.TinyGoUID, bkyc.TinyGoGID).
			ImportEnvVars(envVars).
			SetEnvVar("CLI_APP_NAME", cmd.CliName)
	}

	t.Run("happy path", func(t *testing.T) {
		projectToBuild := "hello-world-hash-unvendored-go"
		projectDir := filepath.Join(srcCodeDir, projectToBuild)

		prepareEnv(projectDir).
			CopyDir("in").
			CopyDir("want").
			RunScript(filepath.Join(scriptDir, projectToBuild, "build-happy-path.txtar"))
	})

	t.Run("state lifecycle is managed", func(t *testing.T) {
		projectToBuild := "hello-world-hash-unvendored-go"
		projectDir := filepath.Join(srcCodeDir, projectToBuild)

		prepareEnv(projectDir).
			CopyDir("in").
			CopyDir("want").
			CopyDirToAppStateDir(filepath.Join(
				"state",
				"stale",
				"-1.1d8b0878-b4d7-4bdc-bc1d-2c08ce271109"),
			).
			RunScript(filepath.Join(scriptDir, projectToBuild, "build-state-mgmt.txtar"))
	})

	t.Run("missing args", func(t *testing.T) {
		projectToBuild := "hello-world-hash-unvendored-go"
		projectDir := filepath.Join(srcCodeDir, projectToBuild)

		prepareEnv(projectDir).
			CopyDir("in").
			CopyDir("want").
			RunScript(filepath.Join(scriptDir, projectToBuild, "build-missing-args.txtar"))
	})

	t.Run("canceling is handled", func(t *testing.T) {
		projectToBuild := "hello-world-hash-unvendored-go"
		projectDir := filepath.Join(srcCodeDir, projectToBuild)

		prepareEnv(projectDir).
			CopyDir("in").
			CopyDir("want").
			CopyDirToAppStateDir(filepath.Join(
				"state",
				"stale",
				"-1.1d8b0878-b4d7-4bdc-bc1d-2c08ce271109"),
			).
			RunScript(filepath.Join(scriptDir, projectToBuild, "build-cancel.txtar"))
	})

}
