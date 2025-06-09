package integration_test

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/compiler/internal/bkyc"
)

type CliErrOutput string
type CliStdOutput string

func run(
	cmd string,
	args string,
) (
	CliStdOutput,
	CliErrOutput,
	error,
) {
	wCmd := fmt.Sprintf("%s %s", cmd, args)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cliProc := exec.Command("bash", "-c", wCmd)
	cliProc.Stdout = &stdout
	cliProc.Stderr = &stderr

	if err := cliProc.Run(); err != nil {
		errOut := errors.New(stderr.String())
		werr := fmt.Errorf("running command: %w", err)
		return "", "", errors.Join(errOut, werr)
	}

	errOutput := CliErrOutput(stderr.Bytes())
	stdOutput := CliStdOutput(stdout.Bytes())

	return stdOutput, errOutput, nil
}

func RunCLI(args string) (CliStdOutput, CliErrOutput, error) {
	return run(*cliCmd, args)
}

func TestCLIBuild(t *testing.T) {
	t.Cleanup(func() {
		removeContainersByPrefix(bkyc.BkycPrefix)
	})

	t.Run("happy path", func(t *testing.T) {
		// given
		projRootDir := t.TempDir()
		copyTestData(
			t,
			"./testdata/hello-world-hash-unvendored-go/in",
			projRootDir,
		)

		outDir := createTempDir(t, 0777)

		inPath := filepath.Join(projRootDir, "main.go")
		outPath := filepath.Join(outDir, "got.wasm")

		// when
		args := fmt.Sprintf(
			"build %s %s",
			inPath,
			outPath,
		)
		_, _, gotErr := RunCLI(args)

		// then
		require.NoError(t, gotErr)
		assertFilesEqual(
			t,
			"./testdata/hello-world-hash-unvendored-go/out/want.wasm",
			outPath,
		)
	})

	for name, tc := range map[string]struct {
		args      string
		wantError string
	}{
		"one argument missing": {
			args:      "build arg1",
			wantError: "accepts 2 arg(s), received 1",
		},
		"two arguments missing": {
			args:      "build",
			wantError: "accepts 2 arg(s), received 0",
		},
	} {
		t.Run(name, func(t *testing.T) {
			// when
			_, _, gotErr := RunCLI(tc.args)

			// then
			require.Error(t, gotErr)
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}
}
