package bkyc_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/blocky/compiler/internal/bkyc"
	"github.com/blocky/compiler/internal/container"
)

func TestNewFastGoContainerCfg(t *testing.T) {
	t.Run("happy path - cache", func(t *testing.T) {
		wantCfg := container.Config{
			Image: "tinygo/tinygo@sha256:79c73c4246a1079e648d6e8425bae4654f26d2baed24c96537a8893b41c04d26",
			User:  "tinygo",
			Env: []string{
				"GOCACHE=/tmp/gocache",
				"SOURCE_DATE_EPOCH=1234567890",
			},
			Cmd: []string{
				"sh",
				"-c",
				"sudo chown -R 1000:1000 /home/tinygo/.cache &&" +
					"sudo chown -R 1000:1000 /home/tinygo/go &&" +
					"tinygo build " +
					"-target=wasi " +
					"-o /out/outFile " +
					"-scheduler=none " +
					"-no-debug " +
					"-opt=z " +
					"inFile " +
					"&& touch -d \"@1234567890\" /out/outFile",
			},
			WorkingDir: "/src",
			Tmpfs:      map[string]string{"/tmp": ""},
			Binds: []string{
				"inDir:/src",
				"outDir:/out",
				"cacheDir:/home/tinygo/.cache",
				"goDir:/home/tinygo/go",
			},
			AutoRemove: false,
		}

		// when
		gotCfg := bkyc.NewFastGoContainerCfg(
			"cacheDir",
			"goDir",
			"inDir",
			"inFile",
			"outDir",
			"outFile",
		)

		// then
		gotCfgNoName := gotCfg
		gotCfgNoName.Name = ""
		assert.Equal(t, wantCfg, gotCfgNoName)
		assert.True(t, strings.HasPrefix(gotCfg.Name, bkyc.BkycPrefix))
	})
}

func TestNewReproducibleGoContainerCfg(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		wantCfg := container.Config{
			Image: "tinygo/tinygo@sha256:79c73c4246a1079e648d6e8425bae4654f26d2baed24c96537a8893b41c04d26",
			User:  "tinygo",
			Env: []string{
				"GOCACHE=/tmp/gocache",
				"SOURCE_DATE_EPOCH=1234567890",
			},
			Cmd: []string{
				"sh",
				"-c",
				"tinygo build " +
					"-target=wasi " +
					"-o /out/outFile " +
					"-scheduler=none " +
					"-no-debug " +
					"-opt=z " +
					"inFile " +
					"&& touch -d \"@1234567890\" /out/outFile",
			},
			WorkingDir: "/src",
			Tmpfs:      map[string]string{"/tmp": ""},
			Binds:      []string{"inDir:/src", "outDir:/out"},
			AutoRemove: false,
		}

		// when
		gotCfg := bkyc.NewReproducibleGoContainerCfg(
			"inDir",
			"inFile",
			"outDir",
			"outFile",
		)

		// then
		gotCfgNoName := gotCfg
		gotCfgNoName.Name = ""
		assert.Equal(t, wantCfg, gotCfgNoName)
		assert.True(t, strings.HasPrefix(gotCfg.Name, bkyc.BkycPrefix))
	})
}
