#!/usr/bin/env bash

# Exit on error
set -e

REPO="compiler"
APP="bky-c"

# print script usage help
function printUsage() {
    local name=$1
    echo "Install script usage: ${name} [-v version]" >&2
}

# get the version to download
function getCLIVersion() {
    local opt
    local version="latest"

    while getopts "v:" opt; do
      case "$opt" in
        v)
          version="$OPTARG"
          ;;
        *)
          printUsage $0
          exit 1
          ;;
      esac
    done

    echo "$version"
}

# let the user know a step was successful
function passCheck() {
    echo "✅ $1"
}

# let the user know that we failed with an error
function exitWithErr() {
    echo "❌ ==> $1
       Could not continue.
       For feature requests or support please email info@blocky.rocks." >&2
    exit 1
}

function getOS() {
    case "$OSTYPE" in
        linux*)   echo "linux" ;;
        darwin*)  echo "darwin" ;;
        *)        exitWithErr "Unsupported OS" ;;
    esac
}

function getArch() {
    case "$(uname -m)" in
        x86_64)             echo "amd64" ;;
        arm64 | aarch64)    echo "arm64" ;;
        *)                  exitWithErr "Unsupported architecture" ;;
    esac
}

# check that the os arch combo that the person is installing is supported
function verifySupport() {
    local os=$1
    local arch=$2

    local supported=(linux-amd64 linux-arm64 darwin-amd64 darwin-arm64)
    local current="$os-$arch"

    for i in "${supported[@]}"; do
        if [ "$i" == "$current" ]; then
            passCheck "Your platform is supported: $current"
            return 0
        fi
    done

    printf -v msg \
        'Your platform (%s) is unsupported. Supported platforms are:\n%s' \
        "${current}" \
        "$(printf '       - %s\n' ${supported[@]})"
    exitWithErr "$msg"
}

function verifyCurl() {
    if command -v "curl" > /dev/null; then
        passCheck "You have curl installed: $(which curl)"
    else
        exitWithErr "You do not have curl installed."
    fi
}

function verifyJq() {
    if command -v "jq" > /dev/null; then
        passCheck "You have jq installed: $(which jq)"
    else
        exitWithErr "You do not have jq installed."
    fi
}

function downloadCLI() {
    local version=$1
    local os=$2
    local arch=$3

    if [[ "${version}" == latest ]]; then
      local release=$(curl -s \
                      -H "Accept: application/vnd.github+json" \
                      -H "X-GitHub-Api-Version: 2022-11-28" \
                      "https://api.github.com/repos/blocky/${REPO}/releases" \
                | jq '
                    map(select(.draft == false))
                    | sort_by(.published_at)
                    | reverse
                    | .[0]
                ')
      local artifact="${APP}_${os}_${arch}"
      local url=$(echo "$release" | jq -r --arg name "${artifact}" '
        .assets[] | select(.name == $name) | .browser_download_url
      ')
    else
      local base="https://github.com/blocky/${REPO}/releases/download"
      local artifact="${APP}_${os}_${arch}"
      local url="${base}/${version}/${artifact}"
    fi

    if ! curl --silent --location --fail --show-error "${url}" -o "${APP}"; then
        exitWithErr " CLI download failed"
    fi

    chmod +x "${APP}"
}

function verifyCLI() {
    if ./${APP} --help > /dev/null 2>&1; then
        passCheck "SUCCESS! You have downloaded the ${APP} CLI"
    else
        exitWithErr "install failed"
    fi
}

function main() {
    local os=$(getOS)
    local arch=$(getArch)

    verifySupport "$os" "$arch"
    verifyCurl
    verifyJq

    version=$(getCLIVersion "$@")
    echo "Selected CLI version: $version"
    downloadCLI "$version" "$os" "$arch"
    verifyCLI
}

main "$@"
