#!/bin/bash

# Make the script executable
chmod +x ./test.sh

# Ensure dist directory exists
mkdir -p dist

# Detect OS and architecture
OS=$(uname -s)
ARCH=$(uname -m)

# Select the appropriate binary
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
  # Default to linux-amd64 for other platforms
  BINARY="./dist/progzer-linux-amd64"
  STAT_CMD="stat -c%s"
fi

echo "Using binary: $BINARY"

# Test with a known size file (create a 100MB test file)
echo "Creating a 100MB test file..."
dd if=/dev/zero of=test_file bs=1M count=100

# Get file size using the appropriate stat command
FILE_SIZE=$($STAT_CMD test_file)

# Test with gzip (compressing the test file)
echo -e "\nTest with gzip (known size):"
$BINARY --size=$FILE_SIZE <test_file | gzip >test_file.gz

# Test without specifying size (indeterminate mode)
echo -e "\nTest without size (indeterminate mode):"
cat test_file | $BINARY >/dev/null

# Test with a pipeline of commands
echo -e "\nTest with a pipeline of commands:"
cat test_file | $BINARY | gzip | $BINARY >/dev/null

# Test version flag
echo -e "\nTesting version flag:"
$BINARY --version

# Clean up
echo -e "\nCleaning up..."
rm -f test_file test_file.gz

echo -e "\nTests completed!"
