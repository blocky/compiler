package integration

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/bkyc/internal/bkyc"
	"github.com/blocky/bkyc/internal/container"
)

func copyTestData(t *testing.T, srcPath string, dstPath string) {
	err := copy.Copy(srcPath, dstPath)
	require.NoError(t, err)
}

func assertFilesEqual(t *testing.T, want string, got string) {
	wantBin, err := os.ReadFile(want)
	assert.NoError(t, err)

	gotBin, err := os.ReadFile(got)
	assert.NoError(t, err)

	assert.True(
		t,
		bytes.Equal(wantBin, gotBin),
		"file bytes of '%s' and '%s' should be equal",
		want,
		got,
	)
}

func assertDirEmpty(t *testing.T, path string) {
	content, err := os.ReadDir(path)
	require.NoError(t, err)
	assert.Empty(t, content, "dir '%s' should be empty", path)
}

func removeContainersByPrefix(prefix string) *exec.Cmd {
	removeCmd := fmt.Sprintf(`
        ids=$(docker ps -a --filter "name=^/%s" -q)
        if [ -n "$ids" ]; then
            docker rm -f $ids
        fi
    `, prefix)
	return exec.Command("bash", "-c", removeCmd)
}

func Test_CompileGo(t *testing.T) {
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

		inPath := filepath.Join(projRootDir, "main.go")
		outPath := filepath.Join(t.TempDir(), "got.wasm")

		// when
		gotErr := bkyc.CompileGo(
			context.Background(),
			container.NewRuntime(slog.Default()),
			inPath,
			outPath,
		)

		// then
		assert.NoError(t, gotErr)
		assertFilesEqual(
			t,
			"./testdata/hello-world-hash-unvendored-go/out/want.wasm",
			outPath,
		)
	})

	t.Run("incorrect source code", func(t *testing.T) {
		// given
		projRootDir := t.TempDir()
		copyTestData(
			t,
			"./testdata/incorrect-source-go/in",
			projRootDir,
		)

		outDir := t.TempDir()

		inPath := filepath.Join(projRootDir, "main.go")
		outPath := filepath.Join(outDir, "got.wasm")

		// when
		gotErr := bkyc.CompileGo(
			context.Background(),
			container.NewRuntime(slog.Default()),
			inPath,
			outPath,
		)

		// then
		require.Error(t, gotErr)

		assert.ErrorContains(t, gotErr, "running compilation")
		assert.ErrorContains(t, gotErr, "running container")
		assert.ErrorContains(t, gotErr, "status: '1'")
		assert.ErrorContains(t, gotErr, "main.go:17:7: expected ';', found file")
		assert.ErrorContains(t, gotErr, "main.go:18:3: expected '}', found 'EOF'")

		assertDirEmpty(t, outDir)
	})

	t.Run("missing input file", func(t *testing.T) {
		// given
		projRootDir := t.TempDir()
		outDir := t.TempDir()

		copyTestData(t, "./testdata/no-main-file-go/in", projRootDir)

		inPath := filepath.Join(projRootDir, "main.go")
		outPath := filepath.Join(outDir, "got.wasm")

		// when
		gotErr := bkyc.CompileGo(
			context.Background(),
			container.NewRuntime(slog.Default()),
			inPath,
			outPath,
		)

		// then
		require.Error(t, gotErr)

		assert.ErrorContains(t, gotErr, "creating go compilation config")
		assert.ErrorContains(t, gotErr, "could not find project root")
		assert.ErrorContains(t, gotErr, "main.go: no such file or directory")

		assertDirEmpty(t, outDir)
	})

	t.Run("no project root found", func(t *testing.T) {
		// given
		projRootDir := t.TempDir()
		outDir := t.TempDir()

		copyTestData(t, "./testdata/no-mod-file-go/in", projRootDir)

		inPath := filepath.Join(projRootDir, "main.go")
		outPath := filepath.Join(outDir, "got.wasm")

		// when
		gotErr := bkyc.CompileGo(
			context.Background(),
			container.NewRuntime(slog.Default()),
			inPath,
			outPath,
		)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, "creating go compilation config")
		assert.ErrorContains(t, gotErr, "could not find project root")
		assertDirEmpty(t, outDir)
	})

	t.Run("output folder does not exist", func(t *testing.T) {
		// given
		projRootDir := t.TempDir()
		copyTestData(t, "./testdata/hello-world-hash-unvendored-go/in", projRootDir)

		inPath := filepath.Join(projRootDir, "main.go")
		outPath := filepath.Join("./no-such-folder-exists", "got.wasm")

		// when
		gotErr := bkyc.CompileGo(
			context.Background(),
			container.NewRuntime(slog.Default()),
			inPath,
			outPath,
		)

		// then
		assert.Error(t, gotErr)

		assert.ErrorContains(t, gotErr, "creating go compilation config")
		assert.ErrorContains(t, gotErr, "output folder does not exist")
		assert.ErrorContains(t, gotErr, "no-such-folder-exists/got.wasm")
	})
}
