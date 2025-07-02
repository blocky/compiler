package bkyc

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/blocky/compiler/internal/container"
)

const (
	// TinyGo hash points to tinygo/0.34.0
	TinyGo               = "tinygo/tinygo@sha256:79c73c4246a1079e648d6e8425bae4654f26d2baed24c96537a8893b41c04d26"
	TinyGoGID            = 1000
	TinyGoUID            = 1000
	BkycPrefix           = "bkyc-tinygo"
	FixedSourceDateEpoch = "1234567890"
)

func NewFastGoContainerCfg(
	cacheDir string,
	goDir string,
	inDir string,
	inFile string,
	outDir string,
	outFile string,
) container.Config {
	return container.Config{
		Image: TinyGo,
		Name: fmt.Sprintf(
			"%s-%d-%s",
			BkycPrefix,
			time.Now().Unix(),
			uuid.New().String(),
		),
		User: "tinygo",
		Env: []string{
			"GOCACHE=/tmp/gocache",
			fmt.Sprintf("SOURCE_DATE_EPOCH=%s", FixedSourceDateEpoch),
		},
		WorkingDir: "/src",
		Cmd: []string{
			"sh", "-c",
			fmt.Sprintf(
				`sudo chown -R %v:%v /home/tinygo/.cache &&`+
					`sudo chown -R %v:%v /home/tinygo/go &&`+
					`tinygo build `+
					`-target=wasi `+
					`-o /out/%s `+
					`-scheduler=none `+
					`-no-debug `+
					`-opt=z `+
					`%s `+
					`&& touch -d "@%s" /out/%s`,
				TinyGoUID,
				TinyGoGID,
				TinyGoUID,
				TinyGoGID,
				outFile,
				inFile,
				FixedSourceDateEpoch,
				outFile,
			),
		},
		Tmpfs: map[string]string{"/tmp": ""},
		Binds: []string{
			fmt.Sprintf("%s:/src", inDir),
			fmt.Sprintf("%s:/out", outDir),
			fmt.Sprintf("%s:/home/tinygo/.cache", cacheDir),
			fmt.Sprintf("%s:/home/tinygo/go", goDir),
		},
		AutoRemove: false,
	}
}

func NewReproducibleGoContainerCfg(
	inDir string,
	inFile string,
	outDir string,
	outFile string,
) container.Config {
	return container.Config{
		Image: TinyGo,
		Name: fmt.Sprintf(
			"%s-%d-%s",
			BkycPrefix,
			time.Now().Unix(),
			uuid.New().String(),
		),
		User: "tinygo",
		Env: []string{
			"GOCACHE=/tmp/gocache",
			fmt.Sprintf("SOURCE_DATE_EPOCH=%s", FixedSourceDateEpoch),
		},
		WorkingDir: "/src",
		Cmd: []string{
			"sh", "-c",
			fmt.Sprintf(
				`tinygo build `+
					`-target=wasi `+
					`-o /out/%s `+
					`-scheduler=none `+
					`-no-debug `+
					`-opt=z `+
					`%s `+
					`&& `+
					`touch -d "@%s" /out/%s`,
				outFile,
				inFile,
				FixedSourceDateEpoch,
				outFile,
			),
		},
		Tmpfs: map[string]string{"/tmp": ""},
		Binds: []string{
			fmt.Sprintf("%s:/src", inDir),
			fmt.Sprintf("%s:/out", outDir),
		},
		AutoRemove: false,
	}
}

func NewGoCompilerCfg(
	inPath string,
	outPath string,
	reproducible bool,
) (container.Config, error) {
	zeroRet := container.Config{}
	inDir, inFile, err := NormalizeInput(inPath)
	if err != nil {
		return zeroRet, fmt.Errorf("normalizing input paths: %w", err)
	}
	outDir, outFile, err := NormalizeOutput(outPath)
	if err != nil {
		return zeroRet, fmt.Errorf("normalizing output paths: %w", err)
	}

	if reproducible {
		return NewReproducibleGoContainerCfg(inDir, inFile, outDir, outFile), nil
	}

	cachePath, goPath := "", ""
	cacheRoot, err := FindCacheRoot()
	if err != nil {
		return zeroRet, fmt.Errorf("finding cache root: %w", err)
	}
	cachePath = filepath.Join(cacheRoot, "bky-c")
	goPath = filepath.Join(cacheRoot, "go")
	return NewFastGoContainerCfg(cachePath, goPath, inDir, inFile, outDir, outFile), nil
}
