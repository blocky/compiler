> :exclamation: The BLOCKY Compiler is provided with no
> guarantees. By trying out the repo, binary or invoking the `bky-c` executable you
> agree to not hold BLOCKY responsible for any problems, mishaps,
> adverse effects, or frustrations.
> 
# BKY-C
WASM (reproducible) compilation tool

## Installation

To install the latest version run:
```bash
curl -s https://raw.githubusercontent.com/blocky/compiler/refs/heads/main/install.sh | bash
```

To install a specific version run:
```bash
curl -s https://raw.githubusercontent.com/blocky/compiler/refs/heads/main/install.sh | bash -s -- -v <selected-version>
```
for example:
```bash
curl -s https://raw.githubusercontent.com/blocky/compiler/refs/heads/main/install.sh | bash -s -- -v v0.1.0-beta.1

```

## Building go to WASM
To build your `go` application into `WASM` use `bky-c` build command:
```bash
bky-c build <path-to-your-app> <path-to-output-binary-file>
```
for example
```bash
bky-c build ./main.go ./out/x.wasm
```

please note that:

`<path-to-your-app>` can be either relative and absolute and point to:
- an input file, or
- an input/project folder

You also need to make sure thar the folder in which you wish to put
the output binary exists. The compiler will not create the folder for you.

## Viewing licenses
To display all licenses associated with `bky-c` and its dependencies run:
```bash
bky-c licenses
```

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
