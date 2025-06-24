# Crypto Bubble Map Indexer

A microservice that listens to Ethereum transaction events from NATS, processes the data, and stores wallet relationships in Neo4J for blockchain analytics and bubble mapping.

## ✨ Features

- **Real-time Event Processing**: Listens to NATS JetStream for transaction events
- **Graph Database Integration**: Stores wallet relationships in Neo4J
- **Microservice Architecture**: Clean, scalable, and maintainable design
- **Blockchain Analytics**: Tracks wallet interactions and transaction patterns
- **Health Monitoring**: Built-in health checks and monitoring
- **Docker Support**: Full Docker and Docker Compose support
- **Error Recovery**: Automatic reconnection and error recovery

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   Ethereum      │───▶│   Crawler    │───▶│    NATS         │
│   Blockchain    │    │   Service    │    │   JetStream     │
└─────────────────┘    └──────────────┘    └─────────┬───────┘
                                                     │
                                                     ▼
                                              ┌──────────────┐
                                              │   Indexer    │
                                              │   Service    │
                                              └──────┬───────┘
                                                     │
                                                     ▼
                                              ┌──────────────┐
                                              │    Neo4J     │
                                              │   Database   │
                                              └──────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose
- Neo4J Database
- NATS Server (from ethereum-raw-data-crawler)

### Setup

1. **Clone and setup environment:**
   ```bash
   cd crypto-bubble-map-indexer
   make setup
   ```

2. **Configure environment:**
   ```bash
   cp env.example .env
   # Edit .env with your NATS and Neo4J settings
   ```

3. **Start the indexer:**
   ```bash
   # Using Docker (recommended)
   make up

   # Or build and run locally
   make build && make run
   ```

## 📋 Usage

### Using Makefile

```bash
# Build indexer
make build

# Run locally
make run

# Start with Docker Compose
make up

# View logs
make logs

# Stop services
make down

# Check status
make status

# Run tests
make test
```

## ⚙️ Configuration

Key environment variables in `.env`:

```bash
# NATS Configuration
NATS_URL=nats://localhost:4222
NATS_STREAM_NAME=TRANSACTIONS
NATS_SUBJECT_PREFIX=transactions
NATS_CONSUMER_GROUP=bubble-map-indexer
NATS_CONNECT_TIMEOUT=10s
NATS_RECONNECT_ATTEMPTS=5
NATS_RECONNECT_DELAY=2s

# Neo4J Configuration
NEO4J_URI=neo4j://localhost:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=password
NEO4J_DATABASE=neo4j

# Application Configuration
APP_ENV=production
LOG_LEVEL=info
WORKER_POOL_SIZE=10
BATCH_SIZE=100
```

## 📊 Data Model

### Neo4J Graph Schema

#### Nodes
- **Wallet**: Represents Ethereum addresses
  - Properties: `address`, `first_seen`, `last_seen`, `total_transactions`
- **Transaction**: Represents individual transactions
  - Properties: `hash`, `block_number`, `value`, `gas_used`, `timestamp`

#### Relationships
- **SENT_TO**: Wallet → Transaction → Wallet
  - Properties: `value`, `gas_price`, `timestamp`
- **RECEIVED_FROM**: Wallet → Transaction → Wallet
  - Properties: `value`, `gas_price`, `timestamp`

## 🔍 Analytics Queries

### Find Wallet Connections
```cypher
MATCH (w1:Wallet {address: "0x123..."})-[r:SENT_TO]->(w2:Wallet)
RETURN w1.address, w2.address, r.value, r.timestamp
ORDER BY r.timestamp DESC
LIMIT 10
```

### Find Transaction Paths
```cypher
MATCH path = (w1:Wallet)-[:SENT_TO*1..3]->(w2:Wallet)
WHERE w1.address = "0x123..." AND w2.address = "0x456..."
RETURN path
```

### Bubble Analysis
```cypher
MATCH (w:Wallet)-[:SENT_TO]->(other:Wallet)
WITH w, count(other) as connections
WHERE connections > 10
RETURN w.address, connections
ORDER BY connections DESC
```

## 📈 Monitoring

### Health Checks

```bash
# Check service health
curl http://localhost:8080/health

# Check Neo4J connection
curl http://localhost:8080/health/neo4j

# Check NATS connection
curl http://localhost:8080/health/nats
```

### Metrics

The service exposes metrics at `/metrics` endpoint for Prometheus monitoring.

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration
```

## 🐳 Docker

### Development
```bash
docker-compose up -d
```

### Production
```bash
docker-compose -f docker-compose.prod.yml up -d
```

## 🔧 Development

### Project Structure
```
crypto-bubble-map-indexer/
├── cmd/
│   └── indexer/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── entity/
│   │   ├── repository/
│   │   └── service/
│   ├── application/
│   │   ├── service/
│   │   └── usecase/
│   ├── infrastructure/
│   │   ├── config/
│   │   ├── database/
│   │   ├── messaging/
│   │   └── logger/
│   └── adapters/
│       ├── primary/
│       └── secondary/
├── pkg/
├── scripts/
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Data Model Changes (Direct Wallet to Wallet)

As of the latest update, the data model has been simplified to only include Wallet nodes with direct relationships between them:

1. **Wallet Nodes**: Each node represents an Ethereum wallet/address
   - Properties include: address, first_seen, last_seen, total_transactions, total_sent, total_received, network

2. **SENT_TO Relationships**: Direct connections between wallets
   - Summary properties: total_value, tx_count, first_tx, last_tx
   - Detailed transaction history: tx_details (array of transaction objects containing):
     - hash: Transaction hash
     - value: Transaction value
     - timestamp: When the transaction occurred

This simplified model removes the Transaction nodes that were previously present while preserving all transaction details. This approach offers several benefits:

- Cleaner graph visualization
- More straightforward queries
- Better performance for traversals between wallets
- Preserved complete transaction history for analysis

Example of accessing transaction details in Cypher:

```cypher
// Get all transactions between two wallets
MATCH (from:Wallet {address: '0x123...'})-[r:SENT_TO]->(to:Wallet {address: '0x456...'})
RETURN r.tx_details

// Find wallets with high-value individual transactions
MATCH (from:Wallet)-[r:SENT_TO]->(to:Wallet)
UNWIND r.tx_details as tx
WHERE toFloat(tx.value) > 10000000000000000000  // > 10 ETH
RETURN from.address, to.address, tx.hash, tx.value, tx.timestamp
ORDER BY toFloat(tx.value) DESC
LIMIT 10
```

To migrate existing data to this model:

```bash
# Run the cleanup script to convert the database
make cleanup
```

This script will:
1. Convert all existing wallet-transaction-wallet patterns to direct wallet-to-wallet relationships
2. Combine multiple transactions between the same wallets into a single relationship with summarized properties
3. Preserve all transaction details in the tx_details array
4. Remove all Transaction nodes from the database