package compatibility

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/blocky/compiler/cmd/bky-c/cmd"
	"github.com/blocky/compiler/internal/bkyc"
	"github.com/blocky/compiler/test"
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

func TestASCompatibility(t *testing.T) {
	t.Cleanup(func() {
		test.RemoveContainersByPrefix(bkyc.BkycPrefix)
	})

	prepareEnv := func(projectDir string) *test.ProjectTest {
		envVars := []string{
			"DOCKER_HOST",
		}
		return test.NewProjectTest(t, projectDir).
			BuildIfMissing(CLIPath(), cmd.CliName).
			DownloadSupportedASReleases().
			MakeDir("xdgHome").
			SetXdgStateHomeDir("xdgHome").
			MakeDir("got").
			ChownIf(runsAsRoot, "got", tinyGoUID, tinyGoGID).
			ImportEnvVars(envVars)
	}

	t.Run("happy path", func(t *testing.T) {
		projectToBuild := "random-example-go"
		projectDir := filepath.Join(srcCodeDir, projectToBuild)

		prepareEnv(projectDir).
			CopyDir("compile-in").
			CopyDir("attest-in").
			RunScript(filepath.Join(scriptDir, projectToBuild, "as-compatibility-happy-path.txtar"))
	})

}
