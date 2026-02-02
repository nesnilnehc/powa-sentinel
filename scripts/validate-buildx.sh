#!/bin/bash

# Validation script for Docker Buildx multi-architecture setup
# This script validates that the Docker Buildx configuration supports multi-architecture builds

set -e

echo "🔍 Validating Docker Buildx multi-architecture setup..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed or not in PATH"
    exit 1
fi

# Check if buildx is available
if ! docker buildx version &> /dev/null; then
    echo "❌ Docker Buildx is not available"
    exit 1
fi

echo "✅ Docker and Buildx are available"

# Create a test builder if it doesn't exist
BUILDER_NAME="multiarch-test-builder"

# Remove existing builder if it exists
docker buildx rm "$BUILDER_NAME" 2>/dev/null || true

# Create new builder with multi-arch support
echo "🔧 Creating multi-architecture builder..."
docker buildx create --name "$BUILDER_NAME" --driver docker-container --platform linux/amd64,linux/arm64 --use

# Inspect the builder to verify platforms
echo "🔍 Inspecting builder capabilities..."
PLATFORMS=$(docker buildx inspect --bootstrap | grep "Platforms:" | cut -d: -f2 | tr -d ' ')

if [[ "$PLATFORMS" == *"linux/amd64"* ]] && [[ "$PLATFORMS" == *"linux/arm64"* ]]; then
    echo "✅ Builder supports required platforms: $PLATFORMS"
else
    echo "❌ Builder does not support required platforms. Found: $PLATFORMS"
    docker buildx rm "$BUILDER_NAME" 2>/dev/null || true
    exit 1
fi

# Test build for both architectures (dry run)
echo "🧪 Testing multi-architecture build (dry run)..."
if docker buildx build --platform linux/amd64,linux/arm64 --dry-run . > /dev/null 2>&1; then
    echo "✅ Multi-architecture build test passed"
else
    echo "❌ Multi-architecture build test failed"
    docker buildx rm "$BUILDER_NAME" 2>/dev/null || true
    exit 1
fi

# Clean up test builder
echo "🧹 Cleaning up test builder..."
docker buildx rm "$BUILDER_NAME" 2>/dev/null || true

echo "🎉 Docker Buildx multi-architecture setup validation completed successfully!"
echo ""
echo "Summary:"
echo "- Docker Buildx is properly configured"
echo "- Multi-architecture support is available (linux/amd64, linux/arm64)"
echo "- Build process can target both architectures"
echo ""
echo "The GitHub Actions workflow should work correctly with this configuration."