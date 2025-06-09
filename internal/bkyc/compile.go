package bkyc

import (
	"context"
	"fmt"

	"github.com/blocky/compiler/internal/container"
)

type ContainerRunner interface {
	Run(context.Context, container.Config) (container.Output, error)
}

func CompileGo(
	ctx context.Context,
	r ContainerRunner,
	inPath string,
	outPath string,
) error {
	goCfg, err := NewGoCompilerCfg(inPath, outPath)
	if err != nil {
		return fmt.Errorf("creating go compilation config: %w", err)
	}

	_, err = r.Run(ctx, goCfg)
	if err != nil {
		return fmt.Errorf("running compilation: %w", err)
	}
	return nil
}
