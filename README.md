# BKY-C
WASM (reproducible) compilation tool

## Dev Environment
For development and testing you should use the provided `Nix` environment.
To run it:
```bash
nix develop
```

## Working with the containerized environment (docker-in-docker/dind setup)
In order to run or debug tests (integration, compatibility, any ending in `-dind`) in an isolated
docker-in-docker setup you need to enable multiplatform builds via `docker build buildx`.
If you need to enable it you may want to look at the following target:
```bash
make container-setup
```

## Debugging in docker-in-docker setup
To debug a test `Test_MyCode` using a container with a dedicated Docker daemon run:
```bash
make start-debug-env-dind testname=Test_MyCode
```
When you see something similar on your terminal:
```text
Waiting for breakpoint in test: Test_MyCode
API server listening at: [::]:2345
2025-06-08T18:58:43Z warning layer=rpc Listening for remote connections (connections are not authenticated nor encrypted)
```
you can start a test debug run from your IDE. In `Goland` standard `Go Remote` test run configuration will be ok.

## Running integration tests in docker-in-docker setup
To run integration tests inside a container with a dedicated Docker daemon run:
```bash
make test-integration-dind
```

## Running compatibility tests in docker-in-docker setup
To run compatibility tests inside a container with a dedicated Docker daemon run:
```bash
make test-compatibility-dind
```
