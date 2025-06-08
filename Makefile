# We must run tidy first so that we run the rest of the
# steps on the correct dependencies. The order of the others do not matter.
pre-pr: tidy lint test-short test-integration test-integration-dind

tidy:
	@go mod tidy

# Run all the "normal" tests, that is, those that are NOT marked as
# non-short mode.
# All tests that skip in short mode should be explicitly marked with
# their own test rule
test-short:
	@go test -v ./... -race -short -count=1

# All integration tests are skipped when running in short mode.
# They may rely on external services, for example "docker"
test-integration:
	@go test -v ./test/integration/... -race -count=1

# All integration tests but run in a docker-in-docker container
# Allows for isolation between host and guest docker daemons
test-integration-dind:
	docker run --rm \
		--name=test-integration-dind \
		--user=root \
		--privileged \
		-v .:/src \
		-w /src \
		docker:28.2.2-dind-rootless \
		sh ./dind/run-cmd.sh \
		"go test -v ./test/integration/... -race  -count=1"

# Start a dind container with remote delve debugger
start-debug-dind-env:
	docker run --rm -it \
		--name=debug-dind-env \
		--user=root \
		--privileged \
		-p 2345:2345 \
		-v .:/src \
		-w /src \
		docker:28.2.2-dind-rootless \
		sh ./dind/debug-test.sh $(testname)

lint:
	@golangci-lint run --config golangci.yaml

fix-lint:
	@golangci-lint run --config golangci.yaml --fix

mock: tidy
	@rm -rf mocks
	@mockery --quiet --config=mockery.yaml

veryclean:
	@rm -rf mocks
