package container

import (
	"context"
	"fmt"
)

const OK = 0

type Client interface {
	Compatible(context.Context) (bool, error)
	ImageExists(context.Context, string) (bool, error)
	PullImage(context.Context, string) error
	Create(context.Context, Config) (string, error)
	Start(context.Context, string) error
	Wait(context.Context, string) (int, string, error)
	Logs(context.Context, string) (string, error)
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
		return err
	}
	if !isLocal {
		r.log.Debug("Image not available, pulling", "image", image)
		err := r.client.PullImage(ctx, image)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) StartContainer(
	ctx context.Context,
	cfg Config,
) (string, error) {
	cID, err := r.client.Create(ctx, cfg)
	if err != nil {
		return "", err
	}
	if err := r.client.Start(ctx, cID); err != nil {
		return "", err
	}
	return cID, nil
}

type Output struct {
	Status   int
	ErrorMsg string
	Logs     string
}

func (r *Runtime) GetContainerOutput(
	ctx context.Context,
	cID string,
) (Output, error) {
	zeroRet := Output{}
	status, errMsg, err := r.client.Wait(ctx, cID)
	if err != nil {
		return zeroRet, err
	}
	logs, err := r.client.Logs(ctx, cID)
	if err != nil {
		return zeroRet, err
	}
	return Output{
		Status:   status,
		ErrorMsg: errMsg,
		Logs:     logs,
	}, nil
}

func (r *Runtime) CleanUpContainer(
	ctx context.Context,
	cID string,
) error {
	if err := r.client.Stop(ctx, cID); err != nil {
		r.log.Debug("stopping container", "err", err)
	}
	if err := r.client.Remove(ctx, cID); err != nil {
		return err
	}
	return nil
}

func (r *Runtime) Run(
	ctx context.Context,
	cfg Config,
) (Output, error) {
	zeroRet := Output{}
	ok, err := r.Compatible(ctx)
	if err != nil {
		return zeroRet, err
	}
	if !ok {
		return zeroRet, fmt.Errorf("runtime not compatible")
	}

	err = r.GetImage(ctx, cfg.Image)
	if err != nil {
		return zeroRet, err
	}

	cID, err := r.StartContainer(ctx, cfg)
	if err != nil {
		return zeroRet, err
	}
	defer func() {
		err := r.CleanUpContainer(ctx, cID)
		if err != nil {
			r.log.Warn("cleaning up container", "err", err)
		}
	}()

	output, err := r.GetContainerOutput(ctx, cID)
	if err != nil {
		return zeroRet, err
	}

	r.log.Debug(output.Logs)
	if output.Status != OK {
		return Output{}, fmt.Errorf(
			"running container, status: '%d', msg: '%s', log: '%s'",
			output.Status,
			output.ErrorMsg,
			output.Logs,
		)
	}
	return output, nil
}
