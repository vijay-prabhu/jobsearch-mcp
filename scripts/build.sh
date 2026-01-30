#!/bin/bash
# Build script for jobsearch-mcp

set -e

VERSION=${1:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS="-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME"

echo "Building jobsearch-mcp..."
echo "  Version: $VERSION"
echo "  Commit: $COMMIT"
echo "  Build Time: $BUILD_TIME"

# Build for current platform
go build -ldflags "$LDFLAGS" -o bin/jobsearch cmd/jobsearch/main.go

echo ""
echo "Build complete: bin/jobsearch"
