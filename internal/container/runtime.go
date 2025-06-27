package container

import (
	"context"
	"fmt"
)

const OK = 0

type Client interface {
	Compatible(context.Context) (bool, error)
	ImageExists(context.Context, string) (bool, error)
	ContainerExists(context.Context, string) (bool, error)
	PullImage(context.Context, string) error
	Create(context.Context, Config) (string, error)
	Start(context.Context, string) error
	Wait(context.Context, string) (int, string, error)
	Logs(context.Context, string) (string, string, error)
	Stop(context.Context, string) error
	Remove(context.Context, string) error
}

func NewRuntimeFromRaw(client Client, log Logger) *Runtime {
	return &Runtime{
		client: client,
		log:    log,
	}
}

func NewRuntime(log Logger) *Runtime {
	return &Runtime{
		client: NewAPIClientWithLogger(log),
		log:    log,
	}
}

type Runtime struct {
	client Client
	log    Logger
}

func (r *Runtime) Compatible(ctx context.Context) (bool, error) {
	return r.client.Compatible(ctx)
}

func (r *Runtime) GetImage(
	ctx context.Context,
	image string,
) error {
	isLocal, err := r.client.ImageExists(ctx, image)
	if err != nil {
		return fmt.Errorf("checking image '%s': %w", image, err)
	}
	if !isLocal {
		r.log.Debug("Image not available, pulling", "image", image)
		err := r.client.PullImage(ctx, image)
		if err != nil {
			return fmt.Errorf("pulling image '%s': %w", image, err)
		}
	}
	return nil
}

func (r *Runtime) Launch(
	ctx context.Context,
	cfg Config,
) (string, error) {
	cID, err := r.client.Create(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}
	if err := r.client.Start(ctx, cID); err != nil {
		return "", fmt.Errorf("starting container: %w", err)
	}
	return cID, nil
}

type Output struct {
	Status   int
	ErrorMsg string
	StdOut   string
	StdErr   string
}

func (r *Runtime) GetOutput(
	ctx context.Context,
	cID string,
) (Output, error) {
	zeroRet := Output{}
	status, errMsg, err := r.client.Wait(ctx, cID)
	if err != nil {
		return zeroRet, fmt.Errorf("waiting for container '%s': %w", cID, err)
	}
	stdout, stderr, err := r.client.Logs(ctx, cID)
	if err != nil {
		return zeroRet, fmt.Errorf("getting container logs: %w", err)
	}

	return Output{
		Status:   status,
		ErrorMsg: errMsg,
		StdOut:   stdout,
		StdErr:   stderr,
	}, nil
}

func (r *Runtime) CleanUp(
	ctx context.Context,
	cID string,
) error {
	exists, err := r.client.ContainerExists(ctx, cID)
	switch {
	case err != nil:
		return fmt.Errorf("checking container status '%s': %w", cID, err)
	case !exists:
		r.log.Debug("Container doesn't exist", "id", cID)
		return nil
	}
	if err := r.client.Stop(ctx, cID); err != nil {
		r.log.Debug("stopping container", "err", err.Error())
	}
	if err := r.client.Remove(ctx, cID); err != nil {
		return fmt.Errorf("removing container: %w", err)
	}
	return nil
}

func (r *Runtime) Run(
	ctx context.Context,
	cfg Config,
) (Output, error) {
	zeroRet := Output{}
	ok, err := r.Compatible(ctx)
	switch {
	case err != nil:
		return zeroRet, fmt.Errorf("checking compatibility: %w", err)
	case !ok:
		return zeroRet, fmt.Errorf("runtime not compatible")
	}

	err = r.GetImage(ctx, cfg.Image)
	if err != nil {
		return zeroRet, fmt.Errorf("getting image '%s': %w", cfg.Image, err)
	}

	cID, err := r.Launch(ctx, cfg)
	if err != nil {
		return zeroRet, fmt.Errorf("launching container: %w", err)
	}
	defer func() {
		if err := r.CleanUp(ctx, cID); err != nil {
			r.log.Warn("cleaning up container", "err", err)
		}
	}()

	output, err := r.GetOutput(ctx, cID)
	if err != nil {
		return zeroRet, fmt.Errorf("getting container output: %w", err)
	}

	r.log.Debug(output.StdOut)
	r.log.Debug(output.StdErr)
	if output.Status != OK {
		return zeroRet, fmt.Errorf(
			"running container, status: '%d', msg: '%s', stderr: '%s'",
			output.Status,
			output.ErrorMsg,
			output.StdErr,
		)
	}
	return output, nil
}
