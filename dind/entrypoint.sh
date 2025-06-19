#!/bin/sh

echo "Running from: $(pwd)"
echo "go: $(which go)"
echo "tests: $(ls -la ./test/integration)"
echo "PATH: ${PATH}"

dockerd --host=${DOCKER_HOST} --experimental > /dockerd.log 2>&1 &
until docker version >/dev/null 2>&1; do
  echo "Waiting for dockerd..."
  sleep 1
done
exec "$@"
