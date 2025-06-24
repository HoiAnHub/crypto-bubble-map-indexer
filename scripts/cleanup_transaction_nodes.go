package main

import (
	"context"
	"crypto-bubble-map-indexer/internal/infrastructure/config"
	"crypto-bubble-map-indexer/internal/infrastructure/database"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/zap"
)

func main() {
	// Setup logger
	log, err := logger.NewLogger("info")
	if err != nil {
		panic(err)
	}
	log = log.WithComponent("cleanup-script")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config", zap.Error(err))
	}

	// Create Neo4j client
	neo4jClient := database.NewNeo4JClient(&cfg.Neo4J, log)

	// Connect to Neo4j
	ctx := context.Background()
	err = neo4jClient.Connect(ctx)
	if err != nil {
		log.Fatal("Failed to connect to Neo4j", zap.Error(err))
	}
	defer neo4jClient.Close(ctx)

	log.Info("Connected to Neo4j, starting cleanup process")

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Create a session
	session := neo4jClient.GetDriver().NewSession(ctxWithTimeout, neo4j.SessionConfig{})
	defer session.Close(ctxWithTimeout)

	// Step 1: Convert relationships to direct wallet-to-wallet
	log.Info("Converting relationships to direct wallet-to-wallet connections")
	convertQuery := `
		MATCH (from:Wallet)-[r:SENT_TO]->(to:Wallet)
		WITH from, to, collect(r) as rels
		MERGE (from)-[newRel:SENT_TO]->(to)
		ON CREATE SET
			newRel.total_value = toString(reduce(s = 0.0, rel IN rels | s + toFloat(rel.value))),
			newRel.tx_count = size(rels),
			newRel.first_tx = reduce(t = datetime(), rel IN rels | CASE WHEN rel.timestamp < t THEN rel.timestamp ELSE t END),
			newRel.last_tx = reduce(t = datetime({year: 1970, month: 1, day: 1}), rel IN rels | CASE WHEN rel.timestamp > t THEN rel.timestamp ELSE t END)
		RETURN count(newRel) as created_relationships
	`

	result, err := session.ExecuteWrite(ctxWithTimeout, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctxWithTimeout, convertQuery, map[string]interface{}{})
	})

	if err != nil {
		log.Error("Failed to convert relationships", zap.Error(err))
		os.Exit(1)
	}

	records := result.(neo4j.ResultWithContext)
	if records.Next(ctxWithTimeout) {
		record := records.Record()
		log.Info("Created direct relationships", zap.Int64("count", record.Values[0].(int64)))
	}

	// Step 2: Delete Transaction nodes
	log.Info("Deleting Transaction nodes")
	deleteQuery := `
		MATCH (t:Transaction)
		DETACH DELETE t
		RETURN count(t) as deleted_transactions
	`

	result, err = session.ExecuteWrite(ctxWithTimeout, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctxWithTimeout, deleteQuery, map[string]interface{}{})
	})

	if err != nil {
		log.Error("Failed to delete Transaction nodes", zap.Error(err))
		os.Exit(1)
	}

	records = result.(neo4j.ResultWithContext)
	if records.Next(ctxWithTimeout) {
		record := records.Record()
		log.Info("Deleted Transaction nodes", zap.Int64("count", record.Values[0].(int64)))
	}

	log.Info("Cleanup complete! Database now only contains Wallet nodes with direct relationships.")
}
