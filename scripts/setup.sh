#!/bin/bash

set -e

echo "🚀 Setting up Crypto Bubble Map Indexer..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.23 or higher."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.23"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or higher."
    exit 1
fi

echo "✅ Go version $GO_VERSION is compatible"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose."
    exit 1
fi

echo "✅ Docker and Docker Compose are available"

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file from template..."
    cp env.example .env
    echo "✅ .env file created. Please review and update the configuration."
else
    echo "✅ .env file already exists"
fi

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download
go mod tidy

# Build the application
echo "🔨 Building the application..."
go build -o bin/indexer cmd/indexer/main.go

echo "✅ Setup completed successfully!"
echo ""
echo "Next steps:"
echo "1. Review and update .env file if needed"
echo "2. Run 'make up' to start the services"
echo "3. Run 'make logs' to view logs"
echo "4. Run 'make down' to stop the services"
echo ""
echo "Access points:"
echo "- Health check: http://localhost:8080/health"
echo "- Neo4J Browser: http://localhost:7474"
echo "- NATS monitoring: http://localhost:8222"