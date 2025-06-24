package bkyc

import (
	"fmt"
	"time"

	"github.com/blocky/compiler/internal/container"
)

const (
	// TinyGo hash points to tinygo/0.34.0
	TinyGo               = "tinygo/tinygo@sha256:79c73c4246a1079e648d6e8425bae4654f26d2baed24c96537a8893b41c04d26"
	BkycPrefix           = "bkyc-tinygo"
	FixedSourceDateEpoch = "1234567890"
)

func NewGoContainerCfg(
	cacheDir string,
	goDir string,
	inDir string,
	inFile string,
	outDir string,
	outFile string,
) container.Config {
	binds := []string{
		fmt.Sprintf("%s:/src", inDir),
		fmt.Sprintf("%s:/out", outDir),
	}
	cmd := fmt.Sprintf(
		`tinygo build `+
			`-target=wasi `+
			`-o /out/%s `+
			`-scheduler=none `+
			`-no-debug `+
			`-opt=z `+
			`%s `+
			`&& touch -d "@%s" /out/%s`,
		outFile,
		inFile,
		FixedSourceDateEpoch,
		outFile,
	)
	if cacheDir != "" {
		binds = append(binds, fmt.Sprintf("%s:/home/tinygo/.cache", cacheDir))
		cmd += `&& sudo chmod -R 0777 /home/tinygo/.cache`
	}
	if goDir != "" {
		binds = append(binds, fmt.Sprintf("%s:/home/tinygo/go", goDir))
		cmd += `&& sudo chmod -R 0777 /home/tinygo/go`
	}

	return container.Config{
		Image: TinyGo,
		Name:  fmt.Sprintf("%s-%d", BkycPrefix, time.Now().Unix()),
		User:  "tinygo",
		Env: []string{
			"GOCACHE=/tmp/gocache",
			fmt.Sprintf("SOURCE_DATE_EPOCH=%s", FixedSourceDateEpoch),
		},
		WorkingDir: "/src",
		Cmd:        []string{"sh", "-c", cmd},
		Tmpfs:      map[string]string{"/tmp": ""},
		Binds:      binds,
		AutoRemove: false,
	}
}

func NewGoCompilerCfg(
	cachePath string,
	goPath string,
	inPath string,
	outPath string,
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
	return NewGoContainerCfg(cachePath, goPath, inDir, inFile, outDir, outFile), nil
}
