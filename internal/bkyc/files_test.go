package bkyc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/bkyc/internal/bkyc"
)

func WriteFile(path string, name string, content []byte) (string, error) {
	fPath := filepath.Join(path, name)
	err := os.WriteFile(
		fPath,
		content,
		0666,
	)
	return fPath, err
}

func WriteNestedFolders(path string, folders ...string) (string, error) {
	foldersPath := filepath.Join(append([]string{path}, folders...)...)
	err := os.MkdirAll(
		foldersPath,
		0777,
	)
	return foldersPath, err
}

func TestSplit(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantFileName := "myfile.txt"
		wantDir := t.TempDir()
		wantFile, err := WriteFile(wantDir, wantFileName, []byte("hello world"))
		require.NoError(t, err)

		// when
		gotDir, gotFile, gotErr := bkyc.Split(wantFile)

		// then
		require.NoError(t, gotErr)
		assert.Equal(t, wantDir, gotDir)
		assert.Equal(t, wantFileName, gotFile)
	})
}

func TestExists(t *testing.T) {
	t.Run("happy path - dir exists", func(t *testing.T) {
		// given
		dir := t.TempDir()

		// when
		ok := bkyc.Exists(dir)

		// then
		assert.True(t, ok)
	})

	t.Run("happy path - dir does not exist", func(t *testing.T) {
		// when
		ok := bkyc.Exists("/no-such-dir")

		// then
		assert.False(t, ok)
	})

	t.Run("happy path - file exists", func(t *testing.T) {
		// given
		dir := t.TempDir()
		file, err := WriteFile(dir, "no-such-file.txt", []byte{})
		require.NoError(t, err)

		// when
		ok := bkyc.Exists(file)

		// then
		assert.True(t, ok)
	})

	t.Run("happy path - file does not exist", func(t *testing.T) {
		// when
		ok := bkyc.Exists("./no-such-file.txt")

		// then
		assert.False(t, ok)
	})
}

func TestNormalizeInput(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		rootDir := t.TempDir()
		_, err := WriteFile(
			rootDir,
			"go.mod",
			[]byte("module my_test_module\n"),
		)
		require.NoError(t, err)

		wantLevels := []string{"level1", "level2"}
		subDir, err := WriteNestedFolders(
			rootDir,
			wantLevels...,
		)
		require.NoError(t, err)

		wantFile := "start.go"
		startFile, err := WriteFile(
			subDir,
			wantFile,
			[]byte("package main\n"),
		)
		require.NoError(t, err)

		// when
		gotDir, gotFile, gotErr := bkyc.NormalizeInput(startFile)

		// then
		require.NoError(t, gotErr)
		assert.Equal(t, rootDir, gotDir)
		assert.True(t, filepath.IsAbs(gotDir))
		assert.Equal(t, filepath.Join(append(wantLevels, wantFile)...), gotFile)
		assert.False(t, filepath.IsAbs(gotFile))
	})

	t.Run("no project root found", func(t *testing.T) {
		// given
		rootDir := t.TempDir()

		subDir, err := WriteNestedFolders(
			rootDir,
			"level1",
			"level2",
		)
		require.NoError(t, err)

		startFile, err := WriteFile(
			subDir,
			"start.go",
			[]byte("package main\n"),
		)
		require.NoError(t, err)

		// when
		_, _, gotErr := bkyc.NormalizeInput(startFile)

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, "could not find project root")
	})
}

func TestNormalizeOutput(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		rootDir := t.TempDir()
		wantLevels := []string{"level1", "level2"}
		subDir, err := WriteNestedFolders(
			rootDir,
			wantLevels...,
		)
		require.NoError(t, err)

		wantFile := "start.go"
		startFile, err := WriteFile(
			subDir,
			wantFile,
			[]byte("package main\n"),
		)
		require.NoError(t, err)

		// when
		gotDir, gotFile, gotErr := bkyc.NormalizeOutput(startFile)

		// then
		require.NoError(t, gotErr)
		assert.Equal(
			t,
			filepath.Join(append([]string{rootDir}, wantLevels...)...),
			gotDir,
		)
		assert.True(t, filepath.IsAbs(gotDir))
		assert.Equal(t, wantFile, gotFile)
		assert.False(t, filepath.IsAbs(gotFile))
	})

	t.Run("output folder does not exist - abs path", func(t *testing.T) {
		// given

		// when
		_, _, gotErr := bkyc.NormalizeOutput("/no-such-dir/no-such-file.txt")

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, "output folder does not exist")
	})

	t.Run("output folder does not exist - rel path", func(t *testing.T) {
		// given

		// when
		_, _, gotErr := bkyc.NormalizeOutput("./no-such-dir/no-such-file.txt")

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, "output folder does not exist")
	})
}

func TestFindGoProjectRoot(t *testing.T) {
	t.Run("happy path - file input", func(t *testing.T) {
		// given
		rootDir := t.TempDir()
		_, err := WriteFile(
			rootDir,
			"go.mod",
			[]byte("module my_test_module\n"),
		)
		require.NoError(t, err)

		subDir, err := WriteNestedFolders(
			rootDir,
			"level1",
			"level2",
		)
		require.NoError(t, err)

		startFile, err := WriteFile(
			subDir,
			"start.go",
			[]byte("package main\n"),
		)
		require.NoError(t, err)

		// when
		gotDir, gotErr := bkyc.FindGoProjectRoot(startFile)

		// then
		require.NoError(t, gotErr)
		require.Equal(t, rootDir, gotDir)
		assert.True(t, filepath.IsAbs(gotDir))
	})

	t.Run("happy path - dir input", func(t *testing.T) {
		// given
		rootDir := t.TempDir()
		_, err := WriteFile(
			rootDir,
			"go.mod",
			[]byte("module my_test_module\n"),
		)
		require.NoError(t, err)

		startDir, err := WriteNestedFolders(
			rootDir,
			"level1",
			"level2",
		)
		require.NoError(t, err)

		// when
		gotDir, gotErr := bkyc.FindGoProjectRoot(startDir)

		// then
		require.NoError(t, gotErr)
		require.Equal(t, rootDir, gotDir)
		assert.True(t, filepath.IsAbs(gotDir))
	})

	t.Run("happy path - nested go.mods", func(t *testing.T) {
		// given
		rootDir := t.TempDir()

		_, err := WriteFile(
			rootDir,
			"go.mod",
			[]byte("module my_test_module_in_root\n"),
		)
		require.NoError(t, err)

		subDir, err := WriteNestedFolders(
			rootDir,
			"level1",
		)
		require.NoError(t, err)

		_, err = WriteFile(
			subDir,
			"go.mod",
			[]byte("module my_test_module_in_sub\n"),
		)
		require.NoError(t, err)

		startFile, err := WriteFile(
			subDir,
			"start.go",
			[]byte("package main\n"),
		)
		require.NoError(t, err)

		// when
		gotDir, gotErr := bkyc.FindGoProjectRoot(startFile)

		// then
		require.NoError(t, gotErr)
		require.Equal(t, subDir, gotDir)
		assert.True(t, filepath.IsAbs(gotDir))
	})

	t.Run("no go.mod", func(t *testing.T) {
		// given
		rootDir := t.TempDir()
		subDir, err := WriteNestedFolders(
			rootDir,
			"level1",
			"level2",
			"level3",
		)
		require.NoError(t, err)

		startFile, err := WriteFile(
			subDir,
			"start.go",
			[]byte("package main\n"),
		)
		require.NoError(t, err)

		// when
		_, gotErr := bkyc.FindGoProjectRoot(startFile)

		// then
		require.ErrorContains(t, gotErr, "cannot find project root")
	})
}
