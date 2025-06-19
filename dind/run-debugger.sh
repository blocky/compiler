#!/bin/sh

TEST_NAME="$1"
if [ -z "$TEST_NAME" ]; then
  echo "test name is missing"
  exit 1
fi

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