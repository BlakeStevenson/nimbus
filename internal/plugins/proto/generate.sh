#!/bin/bash
# Generate protobuf and gRPC code
#
# Prerequisites:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#   brew install protobuf  # or apt-get install -y protobuf-compiler
#
# Run from the repo root:
#   bash internal/plugins/proto/generate.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

cd "$REPO_ROOT"

protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       internal/plugins/proto/plugin.proto

echo "✓ Protocol buffers generated successfully"
