# BKY-C
WASM (reproducible) compilation tool

## Debugging in docker-in-docker setup
To debug a test `Test_MyCode` using a container with a dedicated Docker daemon run:
```bash
make start-debug-dind-env testname=Test_MyCode
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
