package compatibility

import (
	"path/filepath"
	"testing"

	"github.com/blocky/compiler/cmd/bky-c/cmd"
	"github.com/blocky/compiler/internal/bkyc"
	"github.com/blocky/compiler/test"
)

const (
	srcCodeDir = "testdata"
	scriptDir  = "scripts"
)

func CLIPath() string {
	return filepath.Join("..", "..", "cmd", cmd.CliName, "main.go")
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
