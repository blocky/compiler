package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

const empty = ""

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

type Condition func() bool

func (e *ProjectTest) ChownIf(ok Condition, relPath string, uid, gid int) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		if !ok() {
			return nil
		}
		dir := filepath.Join(env.WorkDir, relPath)
		err := os.Chown(dir, uid, gid)
		if err != nil {
			return fmt.Errorf("failed to set owner of directory %s: %w", dir, err)
		}
		return nil
	}
	e.setupFuncs = append(e.setupFuncs, setupFunc)
	return e
}

func (e *ProjectTest) CopyDir(relPath string) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		srcDir := filepath.Join(e.projectDir, relPath)
		return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("failed to walk dir %s: %w", srcDir, err)
			}
			if info.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(filepath.Dir(srcDir), path)
			if err != nil {
				return fmt.Errorf("failed to get relative path of %s: %w", path, err)
			}

			src := filepath.Join(e.projectDir, relPath)
			dst := filepath.Join(env.WorkDir, relPath)
			dstDir := filepath.Dir(dst)
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				msg := "failed to create destination directory %s: %w"
				return fmt.Errorf(msg, dstDir, err)
			}

			srcContent, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("failed to read source file %s: %w", src, err)
			}

			if err := os.WriteFile(dst, srcContent, 0644); err != nil {
				msg := "failed to write destination file %s: %w"
				return fmt.Errorf(msg, dst, err)
			}
			return nil
		})
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

func (e *ProjectTest) ImportEnvVars(envKeys []string, failOnMissing bool) *ProjectTest {
	setupFunc := func(env *testscript.Env) error {
		for _, key := range envKeys {
			val := os.Getenv(key)
			if val == empty && failOnMissing {
				return fmt.Errorf("environment variable %s is not set", key)
			}
			env.Setenv(key, val)
		}
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
		ts.Fatalf("failed to read 'want' file %s: %v", args[0], err)
	}
	got, err := os.ReadFile(ts.MkAbs(args[1]))
	if err != nil {
		ts.Fatalf("failed to read 'got' file %s: %v", args[1], err)
	}
	if bytes.Equal(want, got) == neg {
		msg := map[bool]string{true: "equal", false: "not equal"}[neg]
		ts.Fatalf("file '%s' and file '%s' are '%s'", want, got, msg)
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
		"bin-eq": assertBinEqual,
	}
	e.params.Files = []string{scriptFile}
	testscript.Run(e.t, e.params)
}
