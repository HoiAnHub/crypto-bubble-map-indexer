#!/bin/bash

# VPS Deployment Script for Crypto Bubble Map Indexer
# This script sets up the full stack on a VPS

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VPS_IP="45.149.206.55"
API_PORT="8080"
NEO4J_BROWSER_PORT="7474"
NEO4J_BOLT_PORT="7687"

echo -e "${BLUE}🚀 VPS Deployment Script for Crypto Bubble Map Indexer${NC}"
echo -e "${BLUE}===================================================${NC}"
echo ""

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check prerequisites
echo -e "${BLUE}📋 Checking prerequisites...${NC}"

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

if ! command -v make &> /dev/null; then
    print_error "Make is not installed. Please install Make first."
    exit 1
fi

print_status "All prerequisites are installed"

# Check if .env file exists
if [ ! -f ".env" ]; then
    print_warning ".env file not found. Creating from example..."
    if [ -f "env.example" ]; then
        cp env.example .env
        print_status ".env file created from example"
        print_warning "Please edit .env file with your configuration before continuing"
        echo "Press Enter to continue after editing .env file..."
        read
    else
        print_error "env.example file not found. Please create .env file manually."
        exit 1
    fi
fi

# Check firewall status
echo ""
echo -e "${BLUE}🔥 Checking firewall configuration...${NC}"

# Try to detect firewall type and check port status
if command -v ufw &> /dev/null; then
    if ufw status | grep -q "Status: active"; then
        print_info "UFW firewall is active"
        for PORT in $API_PORT $NEO4J_BROWSER_PORT $NEO4J_BOLT_PORT; do
            if ufw status | grep -q "$PORT"; then
                print_status "Port $PORT is already allowed in UFW"
            else
                print_warning "Port $PORT is not allowed in UFW"
                echo "Run: sudo ufw allow $PORT"
            fi
        done
    else
        print_info "UFW firewall is inactive"
    fi
elif command -v firewall-cmd &> /dev/null; then
    if firewall-cmd --state | grep -q "running"; then
        print_info "firewalld is active"
        for PORT in $API_PORT $NEO4J_BROWSER_PORT $NEO4J_BOLT_PORT; do
            if firewall-cmd --list-ports | grep -q "$PORT"; then
                print_status "Port $PORT is already allowed in firewalld"
            else
                print_warning "Port $PORT is not allowed in firewalld"
                echo "Run: sudo firewall-cmd --permanent --add-port=$PORT/tcp && sudo firewall-cmd --reload"
            fi
        done
    else
        print_info "firewalld is inactive"
    fi
else
    print_warning "No firewall detected. Please ensure ports $API_PORT, $NEO4J_BROWSER_PORT, and $NEO4J_BOLT_PORT are accessible."
fi

# Check network connectivity to NATS
echo ""
echo -e "${BLUE}🔌 Checking NATS connectivity...${NC}"
if nc -z $VPS_IP 4222 &>/dev/null; then
    print_status "NATS is accessible at $VPS_IP:4222"
else
    print_warning "Cannot connect to NATS at $VPS_IP:4222. Make sure NATS is running."
fi

# Deploy the stack
echo ""
echo -e "${BLUE}🚀 Deploying full stack...${NC}"

print_info "Building and starting containers..."
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# Wait for services to be ready
echo ""
print_info "Waiting for services to be ready..."
sleep 20

# Health check
echo ""
echo -e "${BLUE}🏥 Running health check...${NC}"

if curl -s "http://localhost:$API_PORT/health" | grep -q "ok"; then
    print_status "API health check passed"
else
    print_error "API health check failed"
fi

# Check Neo4J
if nc -z localhost $NEO4J_BOLT_PORT &>/dev/null; then
    print_status "Neo4J is running on port $NEO4J_BOLT_PORT"
else
    print_error "Neo4J is not accessible on port $NEO4J_BOLT_PORT"
fi

# Test remote accessibility
if curl -s "http://$VPS_IP:$API_PORT/health" | grep -q "ok"; then
    print_status "API is accessible remotely at http://$VPS_IP:$API_PORT"
else
    print_warning "API is not accessible remotely. Check firewall configuration."
fi

# Final status
echo ""
echo -e "${GREEN}🎉 Deployment completed!${NC}"
echo ""
echo -e "${BLUE}📊 Service URLs:${NC}"
echo -e "  • API Health: ${GREEN}http://$VPS_IP:$API_PORT/health${NC}"
echo -e "  • Neo4J Browser: ${GREEN}http://$VPS_IP:$NEO4J_BROWSER_PORT${NC}"
echo -e "  • Neo4J Bolt: ${GREEN}bolt://$VPS_IP:$NEO4J_BOLT_PORT${NC}"
echo ""
echo -e "${BLUE}🔧 Management Commands:${NC}"
echo -e "  • View logs: ${YELLOW}docker logs crypto-bubble-map-indexer${NC}"
echo -e "  • Neo4J logs: ${YELLOW}docker logs crypto-bubble-map-neo4j${NC}"
echo -e "  • Restart services: ${YELLOW}docker-compose restart${NC}"
echo -e "  • Stop services: ${YELLOW}docker-compose down${NC}"
echo ""
echo -e "${BLUE}🧪 Testing Commands:${NC}"
echo -e "  • Check API health: ${YELLOW}curl http://$VPS_IP:$API_PORT/health${NC}"
echo ""
print_info "You can now access the Neo4J browser from any machine using: http://$VPS_IP:$NEO4J_BROWSER_PORT"
print_info "The indexer is now processing transactions from the NATS stream"