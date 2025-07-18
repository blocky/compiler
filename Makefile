
BKY_DIND_IMAGE_NAME=bky-dind-env-rootless

# We must run tidy first so that we run the rest of the
# steps on the correct dependencies. The order of the others do not matter.
pre-pr: \
 	tidy \
	lint \
	test-short \
	test-integration \
	test-compatibility \
	test-integration-dind \
	test-compatibility-dind

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

# All compatibility tests are skipped when running in short mode.
# They may rely on external services, for example "docker"
# and check inter-operability among various products
# like: bky-c and bky-as
test-compatibility:
	@go test -v ./test/compatibility/... -race -count=1

container-setup:
	docker buildx create --name multiarch-builder --use --driver docker-container
	docker buildx inspect --bootstrap

containers:
	docker buildx build \
		--build-arg GOVERSION=$(shell sed -n 's/^go //p' go.mod) \
		--load \
		--file dind/Dockerfile \
		--platform=$(shell docker info --format '{{.OSType}}')/$(shell uname -m) \
		--tag ${BKY_DIND_IMAGE_NAME}:28.2.2  .

# All integration tests but run in a docker-in-docker container
# Allows for isolation between host and guest docker daemons
test-integration-dind: containers
	docker run --rm \
		--name=test-integration-dind \
		--user=root \
		--privileged \
		-v .:/src \
		-w /src \
		${BKY_DIND_IMAGE_NAME}:28.2.2 \
		go test -v ./test/integration/... -race  -count=1

# All compatibility tests but run in a docker-in-docker container
# Allows for isolation between host and guest docker daemons
test-compatibility-dind: containers
	docker run --rm \
		--name=test-compatibility-dind \
		--user=root \
		--privileged \
		-e BKY_COMPILER_GH_TOKEN \
		-v .:/src \
		-w /src \
		${BKY_DIND_IMAGE_NAME}:28.2.2 \
		go test -v ./test/compatibility/... -race  -count=1

# Start a dind container with remote delve debugger
start-debug-env-dind: containers
	docker run --rm \
		--name=debug-env-dind \
		--user=root \
		--privileged \
		-p 2345:2345 \
		-v .:/src \
		-w /src \
		${BKY_DIND_IMAGE_NAME}:28.2.2 \
		sh ./dind/run-debugger.sh $(testname)

lint:
	@golangci-lint run --config golangci.yaml

fix-lint:
	@golangci-lint run --config golangci.yaml --fix

mock: tidy
	@rm -rf mocks
	@mockery --quiet --config=mockery.yaml

veryclean:
	@rm -rf mocks
