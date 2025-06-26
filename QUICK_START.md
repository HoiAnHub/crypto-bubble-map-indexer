# Crypto Bubble Map Indexer - Quick Start Guide

This guide provides step-by-step instructions to get the Crypto Bubble Map Indexer up and running quickly.

## Prerequisites

- Docker and Docker Compose
- Go 1.21 or later (for local development)
- Access to a NATS JetStream server (typically from ethereum-raw-data-crawler)
- Neo4J database (included in Docker setup)

## Quick Setup with Docker

The easiest way to run the Crypto Bubble Map Indexer is using Docker:

```bash
# Clone the repository
git clone https://github.com/HoiAnHub/crypto-bubble-map-indexer.git
cd crypto-bubble-map-indexer

# Create environment file
cp env.example .env

# Edit the .env file to configure your settings
# Especially update the NATS_URL to point to your NATS server
nano .env

# Start the services
docker-compose up -d
```

## VPS Deployment

For deploying on a VPS, we provide a deployment script:

```bash
# Make the script executable
chmod +x scripts/vps-deploy.sh

# Run the deployment script
./scripts/vps-deploy.sh

# Stop the services
make down
```

## Deployment Result
🎉 Deployment completed!

📊 Service URLs:
  • API Health: http://45.149.206.55:8080/health
  • Neo4J Browser: http://45.149.206.55:7474
  • Neo4J Bolt: bolt://45.149.206.55:7687

🔧 Management Commands:
  • View logs: docker logs crypto-bubble-map-indexer
  • Neo4J logs: docker logs crypto-bubble-map-neo4j
  • Restart services: docker-compose restart
  • Stop services: docker-compose down

🧪 Testing Commands:
  • Check API health: curl http://45.149.206.55:8080/health


The script will:
1. Check prerequisites
2. Configure environment
3. Check firewall settings
4. Deploy the services
5. Perform health checks

## Configuration

Key configuration options in the `.env` file:

```
# NATS Configuration
NATS_URL=nats://ethereum-nats:4222
NATS_STREAM_NAME=TRANSACTIONS
NATS_SUBJECT_PREFIX=transactions
NATS_CONSUMER_GROUP=bubble-map-indexer
NATS_ENABLED=true

# Neo4J Configuration
NEO4J_URI=neo4j://neo4j:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=password
```

## Architecture Overview

The Crypto Bubble Map Indexer consists of:

1. **NATS Consumer**: Subscribes to transaction events from the ethereum-raw-data-crawler
2. **Neo4J Database**: Stores the transaction graph data
3. **Indexing Service**: Processes transactions and creates graph relationships
4. **Health API**: Provides health check endpoints

## Connecting to Ethereum Raw Data Crawler

The indexer connects to the ethereum-raw-data-crawler's NATS server to receive transaction events:

1. Ensure the ethereum-raw-data-crawler is running with NATS enabled
2. Set the `NATS_URL` in your `.env` file to point to the NATS server
3. The indexer will automatically connect and start processing transactions

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

### View Logs

```bash
# View indexer logs
docker logs crypto-bubble-map-indexer

# View Neo4J logs
docker logs crypto-bubble-map-neo4j
```

### Neo4J Browser

Access the Neo4J browser at http://localhost:7474 with default credentials:
- Username: neo4j
- Password: password (or as configured in your .env)

## Common Queries

Once data is indexed, you can run these queries in the Neo4J browser:

```cypher
// View all wallets
MATCH (w:Wallet) RETURN w LIMIT 100;

// View all transactions
MATCH (t:Transaction) RETURN t LIMIT 100;

// Find transactions between wallets
MATCH (from:Wallet)-[r:SENT_TO]->(to:Wallet)
RETURN from.address, to.address, r.value, r.timestamp
LIMIT 100;

// Find the most active wallets
MATCH (w:Wallet)
RETURN w.address, w.total_transactions
ORDER BY w.total_transactions DESC
LIMIT 10;
```

## Troubleshooting

### NATS Connection Issues

If the indexer cannot connect to NATS:

1. Check if NATS is running: `nc -z <nats_host> 4222`
2. Verify the NATS_URL in your .env file
3. Check network connectivity between the indexer and NATS server

### Neo4J Connection Issues

If Neo4J connection fails:

1. Check if Neo4J is running: `docker ps | grep neo4j`
2. Verify Neo4J credentials in your .env file
3. Try connecting manually: `cypher-shell -a neo4j://localhost:7687 -u neo4j -p password`

### Performance Tuning

If you experience performance issues:

1. Increase `WORKER_POOL_SIZE` in docker-compose.yml
2. Increase `NATS_MAX_PENDING_MESSAGES` for larger message buffer
3. Adjust Neo4J memory settings in docker-compose.yml
4. Consider scaling Neo4J for larger datasets

  API is not accessible remotely. Check firewall configuration.

## Next Steps

- Explore the Neo4J browser to visualize the transaction graph
- Connect your frontend applications to the Neo4J database
- Develop custom analytics on the transaction data

For more detailed information, refer to the full README.md file.