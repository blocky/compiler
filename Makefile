# We must run tidy first so that we run the rest of the
# steps on the correct dependencies. The order of the others do not matter.
pre-pr: tidy lint test-short test-integration

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

lint:
	@golangci-lint run --config golangci.yaml

fix-lint:
	@golangci-lint run --config golangci.yaml --fix

mock: tidy
	@rm -rf mocks
	@mockery --quiet --config=mockery.yaml

veryclean:
	@rm -rf mocks