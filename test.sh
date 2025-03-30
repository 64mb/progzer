#!/bin/bash

# exit setup
set -eo pipefail
# [-e] - immediately exit if any command has a non-zero exit status
# [-x] - all executed commands are printed to the terminal [not secure]
# [-o pipefail] - if any command in a pipeline fails, that return code will be used as the return code of the whole pipeline

IS_DONE="false"

echo ""
echo "🧪 Start tests"

report() {
  echo ""

  # clean
  rm -f test_file test_file.gz

  if [ "${IS_DONE}" == "true" ]; then
    echo "✅ tests completed"
    echo ""
    return 0
  fi
  echo "🔴 error in tests"
  echo ""
}
trap 'report' EXIT

echo ""
echo "  checking os and arch..."

OS=$(uname -s)
ARCH=$(uname -m)

echo "  os: ${OS}"
echo "  arch: ${ARCH}"

if [ "$OS" = "Darwin" ] && [ "$ARCH" = "arm64" ]; then
  BINARY="./dist/progzer-darwin-arm64"
  STAT_CMD="stat -f%z"
elif [ "$OS" = "Linux" ]; then
  if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    BINARY="./dist/progzer-linux-arm64"
  else
    BINARY="./dist/progzer-linux-amd64"
  fi
  STAT_CMD="stat -c%s"
else
  # default to linux-amd64 for other platforms
  BINARY="./dist/progzer-linux-amd64"
  STAT_CMD="stat -c%s"
fi

# if exists local binary
if [ -f "./dist/progzer" ]; then
  BINARY="./dist/progzer"
fi

echo "  using binary: ${BINARY}"

echo ""

echo "  creating a random 100MB test file..."
dd if=/dev/urandom of=test_file bs=1M count=100 >/dev/null 2>&1

# Get file size using the appropriate stat command
FILE_SIZE=$(${STAT_CMD} test_file)
FILE_SUM=$(sha256sum test_file | cut -d ' ' -f1)

echo "  file size (bytes): ${FILE_SIZE}"
echo "  file sha256 sum: ${FILE_SUM}"
echo ""

echo "  test with gzip (known size):"
echo ""

${BINARY} --size "${FILE_SIZE}" <test_file | gzip -n >test_file.gz
CHECK_SUM=$(
  # shellcheck disable=SC2094
  ${BINARY} --size "$(${STAT_CMD} test_file.gz)" <test_file.gz |
    gzip -dk |
    sha256sum |
    cut -d ' ' -f1
)

if [ "${CHECK_SUM}" != "${FILE_SUM}" ]; then
  echo "🔴 error: checksum after gzip does not match"
  exit 1
fi

echo ""
echo "  test without size (indeterminate mode):"
echo ""
# shellcheck disable=SC2002
CHECK_SUM=$(cat test_file | ${BINARY} | sha256sum | cut -d ' ' -f1)

if [ "${CHECK_SUM}" != "${FILE_SUM}" ]; then
  echo "🔴 error: checksum without size does not match"
  exit 1
fi

echo ""
echo "  test with a pipeline of commands:"
echo ""
# shellcheck disable=SC2002
FILE_ARCH_SUM=$(cat test_file.gz | sha256sum | cut -d ' ' -f1)
# shellcheck disable=SC2002
CHECK_ARCH_SUM=$(cat test_file | ${BINARY} | gzip -n | ${BINARY} | sha256sum | cut -d ' ' -f1)

echo "  file arch sum: ${FILE_ARCH_SUM}"
echo "  check arch sum: ${CHECK_ARCH_SUM}"

if [ "${CHECK_ARCH_SUM}" != "${FILE_ARCH_SUM}" ]; then
  echo "🔴 error: checksum with a pipeline does not match"
  exit 1
fi

echo ""
echo "  test version flag:"
echo ""

${BINARY} --version

echo ""

IS_DONE="true"
