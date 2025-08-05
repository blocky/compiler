package test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/otiai10/copy"
	"github.com/rogpeppe/go-internal/testscript"
)

const (
	empty = ""
)

type ProjectTest struct {
	t          *testing.T
	params     testscript.Params
	setupFuncs []func(*testscript.Env) error
	projectDir string
}

func NewProjectTest(t *testing.T, projectDir string) *ProjectTest {
	params := testscript.Params{
		Files:               []string{},
		Setup:               nil,
		RequireExplicitExec: true,
		RequireUniqueNames:  true,
	}
	return &ProjectTest{
		t:          t,
		params:     params,
		projectDir: projectDir,
	}
}

func (e *ProjectTest) MakeDir(relPath string) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		dir := filepath.Join(env.WorkDir, relPath)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create a directory %s: %w", dir, err)
		}
		return nil
	}
	e.setupFuncs = append(e.setupFuncs, setupFunc)
	return e
}

func (e *ProjectTest) CopyDir(relPath string) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		srcDir := filepath.Join(e.projectDir, relPath)
		dstDir := filepath.Join(env.WorkDir, relPath)
		return copy.Copy(srcDir, dstDir)
	}
	e.setupFuncs = append(e.setupFuncs, setupFunc)
	return e
}

func (e *ProjectTest) CopyDirToAppStateDir(srcRelPath string) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		srcDir := filepath.Join(e.projectDir, srcRelPath)
		xdgStateHomeDir := env.Getenv("XDG_STATE_HOME")
		if xdgStateHomeDir == "" {
			return fmt.Errorf("XDG_STATE_HOME environment variable not set")
		}
		cliAppName := env.Getenv("CLI_APP_NAME")
		if cliAppName == "" {
			return fmt.Errorf("CLI_APP_NAME environment variable not set")
		}
		dstDir := filepath.Join(xdgStateHomeDir, cliAppName, filepath.Base(srcRelPath))
		return copy.Copy(srcDir, dstDir)
	}
	e.setupFuncs = append(e.setupFuncs, setupFunc)
	return e
}

func isOnPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func addToPath(env *testscript.Env, path string) {
	env.Setenv(
		"PATH",
		filepath.Dir(path)+string(os.PathListSeparator)+env.Getenv("PATH"),
	)
}

func (e *ProjectTest) BuildIfMissing(path string, name string) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		if isOnPath(name) {
			return nil
		}

		outputPath := filepath.Join(env.WorkDir, "bin", name)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputPath, err)
		}
		cmd := exec.Command("go", "build", "-o", outputPath, path)
		cmd.Env = os.Environ()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to build %s: %w", name, err)
		}
		addToPath(env, outputPath)
		return nil
	}

	e.setupFuncs = append(e.setupFuncs, setupFunc)
	return e
}

func (e *ProjectTest) DownloadSupportedASReleases() *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		relInfo, err := GetReleaseInfo(AsRepo, AsSupportedReleases)
		if err != nil {
			return fmt.Errorf("getting release info: %w", err)
		}
		releases, err := DownloadReleases(relInfo, env.WorkDir)
		if err != nil {
			return fmt.Errorf("downloading release info: %w", err)
		}
		for i, release := range releases {
			err := os.Chmod(release.binPath, 0755)
			if err != nil {
				return fmt.Errorf("adding '+x' to binary permissions: %w", err)
			}
			env.Setenv(
				fmt.Sprintf("BKY_AS_RELEASE_%d_BIN", i+1),
				release.binPath,
			)
			env.Setenv(
				fmt.Sprintf("BKY_AS_RELEASE_%d_CONFIG", i+1),
				release.configPath,
			)
		}
		return nil
	}
	e.setupFuncs = append(e.setupFuncs, setupFunc)
	return e
}

func (e *ProjectTest) ImportEnvVars(envKeys []string) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		for _, key := range envKeys {
			val := os.Getenv(key)
			if val == empty {
				continue
			}
			env.Setenv(key, val)
		}
		return nil
	}
	e.setupFuncs = append(e.setupFuncs, setupFunc)
	return e
}

func (e *ProjectTest) SetEnvVar(key, value string) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		env.Setenv(key, value)
		return nil
	}
	e.setupFuncs = append(e.setupFuncs, setupFunc)
	return e
}

func (e *ProjectTest) SetXdgStateHomeDir(path string) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		env.Setenv("XDG_STATE_HOME", filepath.Join(env.WorkDir, path))
		return nil
	}
	e.setupFuncs = append(e.setupFuncs, setupFunc)
	return e
}

func assertBinEqual(ts *testscript.TestScript, neg bool, args []string) {
	expArgs := 2
	if len(args) != expArgs {
		ts.Fatalf("expecting %d args, but got %d", expArgs, len(args))
	}
	want, err := os.ReadFile(ts.MkAbs(args[0]))
	if err != nil {
		ts.Fatalf("reading 'want' file '%s': %v", args[0], err)
	}
	got, err := os.ReadFile(ts.MkAbs(args[1]))
	if err != nil {
		ts.Fatalf("reading 'got' file '%s': %v", args[1], err)
	}
	if bytes.Equal(want, got) == neg {
		msg := map[bool]string{true: "equal", false: "not equal"}[neg]
		ts.Fatalf("file '%s' and file '%s' are '%s'", want, got, msg)
	}
}

func assertDirElementCount(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("negation not supported")
	}
	expArgs := 2
	if len(args) != expArgs {
		ts.Fatalf("expecting %d args, but got %d", expArgs, len(args))
	}
	stat, err := os.Stat(ts.MkAbs(args[0]))
	if err != nil {
		ts.Fatalf("getting stat for '%s': %v", args[0], err)
	}
	if !stat.IsDir() {
		ts.Fatalf("not a directory: '%s'", args[0])
	}
	entries, err := os.ReadDir(ts.MkAbs(args[0]))
	if err != nil {
		ts.Fatalf("reading dir '%s': %v", args[0], err)
	}
	expCount, err := strconv.Atoi(args[1])
	if err != nil {
		ts.Fatalf("expecting numeric value '%s': %v", args[0], err)
	}
	if len(entries) != expCount {
		ts.Fatalf(
			"expecting dir entry count of '%d' to be equal to %d",
			len(entries),
			expCount,
		)
	}
}

func (e *ProjectTest) RunScript(scriptFile string) {
	e.params.Setup = func(env *testscript.Env) error {
		for _, setupFunc := range e.setupFuncs {
			if err := setupFunc(env); err != nil {
				return err
			}
		}
		return nil
	}
	e.params.Cmds = map[string]func(ts *testscript.TestScript, neg bool, args []string){
		"bin-eq":         assertBinEqual,
		"dir-elem-count": assertDirElementCount,
	}
	e.params.Files = []string{scriptFile}
	testscript.Run(e.t, e.params)
}
