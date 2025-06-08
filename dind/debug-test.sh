#!/usr/bin/env bash
set -e

function exitWithErr() {
    echo "❌ ==> $1
       Error occurred." >&2
    exit 1
}

function getGoVersion() {
    sed -n 's/^go //p' go.mod
}

function getOS() {
    local OS=$(uname -s)
    case "${OS}" in
        Linux*)   echo "linux" ;;
        Darwin*)  echo "darwin" ;;
        *)        exitWithErr "Unsupported OS: ${OS}" ;;
    esac
}

function getArch() {
    local ARCH=$(uname -m)
    case "${ARCH}" in
        x86_64)             echo "amd64" ;;
        arm64 | aarch64)    echo "arm64" ;;
        *)                  exitWithErr "Unsupported architecture: ${ARCH}" ;;
    esac
}

TEST_NAME="$1"
if [ -z "$TEST_NAME" ]; then
  exitWithErr "test name is missing"
fi

apk add --no-cache curl delve git tar

GO_BUNDLE="go$(getGoVersion).$(getOS)-$(getArch).tar.gz"
echo "Downloading ${GO_BUNDLE}"

curl --fail --silent --show-error --location "https://go.dev/dl/${GO_BUNDLE}" -o "/tmp/${GO_BUNDLE}"

rm -rf /usr/local/go/
tar -C /usr/local/ -xzf "/tmp/${GO_BUNDLE}"

export GOROOT=/usr/local/go/
export GOPATH=/go
export PATH=${GOROOT}/bin:${GOPATH}/bin:${PATH}

export XDG_RUNTIME_DIR=$(mktemp -d)
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/docker.sock

dockerd --host=$DOCKER_HOST --experimental >dockerd.log 2>&1 &
until docker version >/dev/null 2>&1; do sleep 1; done

INTEGRATION_TEST_OUT=integration-tests-$(date '+%s').test
go test -c -o ${INTEGRATION_TEST_OUT} ./test/integration/...
echo "Waiting for breakpoint in test: ${TEST_NAME}"
dlv exec ${INTEGRATION_TEST_OUT} \
    --listen=:2345 \
    --headless \
    --api-version=2 \
    --accept-multiclient \
    --wd ./test/integration \
    -- -test.run "^${TEST_NAME}$" || true
rm ${INTEGRATION_TEST_OUT}
