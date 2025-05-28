package bkyc

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindGoProjectRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("finding absolute path: %w", err)
	}
	stat, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("getting os stats: %w", err)
	}

	var currPath string
	switch stat.IsDir() {
	case true:
		currPath = absPath
	case false:
		currPath = filepath.Dir(absPath)
	}

	for {
		goModPath := filepath.Join(currPath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currPath, nil
		}
		parentDir := filepath.Dir(currPath)
		if parentDir == currPath {
			return "", fmt.Errorf("cannot find project root")
		}
		currPath = parentDir
	}
}

func RelToRoot(root string, in string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("finding absolute root path: %w", err)
	}
	absIn, err := filepath.Abs(in)
	if err != nil {
		return "", fmt.Errorf("finding absolute input path: %w", err)
	}
	relIn, err := filepath.Rel(absRoot, absIn)
	if err != nil {
		return "", fmt.Errorf("finding relative input path: %w", err)
	}
	return relIn, nil
}

func Split(path string) (string, string, error) {
	abdPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("finding absolute path: %w", err)
	}
	return filepath.Dir(abdPath), filepath.Base(abdPath), nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func NormalizeInput(inPath string) (string, string, error) {
	inDir, err := FindGoProjectRoot(inPath)
	if err != nil {
		return "", "", fmt.Errorf("could not find project root: %w", err)
	}
	inFile, err := RelToRoot(inDir, inPath)
	if err != nil {
		return "", "", fmt.Errorf("finding relative input path: %w", err)
	}
	return inDir, inFile, nil
}

func NormalizeOutput(outPath string) (string, string, error) {
	if !Exists(filepath.Dir(outPath)) {
		return "", "", fmt.Errorf("output folder does not exist: %s", outPath)
	}
	outDir, outFile, err := Split(outPath)
	if err != nil {
		return "", "", fmt.Errorf("splitting output path: %w", err)
	}
	return outDir, outFile, nil
}
