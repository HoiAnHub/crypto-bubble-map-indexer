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

	// Step 1: Extract transaction information and create direct wallet relationships
	log.Info("Extracting transaction information and creating direct wallet relationships")
	extractQuery := `
		// Match transactions and their connected wallets
		MATCH (from:Wallet)-[r:SENT_TO]->(to:Wallet)
		WHERE r.tx_hash IS NOT NULL  // Only process relationships with transaction hashes

		// Group by from and to wallets to consolidate multiple transactions
		WITH from, to, collect(r) as oldRels,
			collect({hash: r.tx_hash, value: r.value, timestamp: r.timestamp}) as tx_details

		// Create or update the new direct relationship
		MERGE (from)-[newRel:WALLET_SENT_TO]->(to)
		SET newRel.total_value = toString(reduce(s = 0.0, rel IN oldRels | s + toFloat(rel.value))),
			newRel.tx_count = size(oldRels),
			newRel.first_tx = reduce(t = datetime(), rel IN oldRels | CASE WHEN rel.timestamp < t THEN rel.timestamp ELSE t END),
			newRel.last_tx = reduce(t = datetime({year: 1970, month: 1, day: 1}), rel IN oldRels | CASE WHEN rel.timestamp > t THEN rel.timestamp ELSE t END),
			newRel.tx_details = tx_details

		RETURN count(newRel) as created_relationships
	`

	result, err := session.ExecuteWrite(ctxWithTimeout, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctxWithTimeout, extractQuery, map[string]interface{}{})
	})

	if err != nil {
		log.Error("Failed to extract transaction information", zap.Error(err))
		os.Exit(1)
	}

	records := result.(neo4j.ResultWithContext)
	if records.Next(ctxWithTimeout) {
		record := records.Record()
		log.Info("Created direct wallet relationships", zap.Int64("count", record.Values[0].(int64)))
	}

	// Step 2: Delete old relationships
	log.Info("Removing old relationships")
	deleteRelsQuery := `
		MATCH (from:Wallet)-[r:SENT_TO]->(to:Wallet)
		WHERE r.tx_hash IS NOT NULL
		DELETE r
		RETURN count(r) as deleted_relationships
	`

	result, err = session.ExecuteWrite(ctxWithTimeout, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctxWithTimeout, deleteRelsQuery, map[string]interface{}{})
	})

	if err != nil {
		log.Error("Failed to delete old relationships", zap.Error(err))
		os.Exit(1)
	}

	records = result.(neo4j.ResultWithContext)
	if records.Next(ctxWithTimeout) {
		record := records.Record()
		log.Info("Deleted old relationships", zap.Int64("count", record.Values[0].(int64)))
	}

	// Step 3: Rename WALLET_SENT_TO relationships to SENT_TO
	log.Info("Renaming WALLET_SENT_TO relationships to SENT_TO")
	renameRelQuery := `
		MATCH (from:Wallet)-[r:WALLET_SENT_TO]->(to:Wallet)
		WITH from, to, properties(r) as props
		MERGE (from)-[newRel:SENT_TO]->(to)
		SET newRel = props
		WITH from, to
		MATCH (from)-[oldRel:WALLET_SENT_TO]->(to)
		DELETE oldRel
		RETURN count(*) as renamed_relationships
	`

	result, err = session.ExecuteWrite(ctxWithTimeout, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctxWithTimeout, renameRelQuery, map[string]interface{}{})
	})

	if err != nil {
		log.Error("Failed to rename relationships", zap.Error(err))
		os.Exit(1)
	}

	records = result.(neo4j.ResultWithContext)
	if records.Next(ctxWithTimeout) {
		record := records.Record()
		log.Info("Renamed relationships to SENT_TO", zap.Int64("count", record.Values[0].(int64)))
	}

	// Step 4: Delete Transaction nodes
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

	// Step 5: Verify final relationships
	log.Info("Verifying final relationships")
	verifyQuery := `
		MATCH ()-[r:SENT_TO]->()
		RETURN count(r) as relationship_count
	`

	result, err = session.ExecuteRead(ctxWithTimeout, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctxWithTimeout, verifyQuery, map[string]interface{}{})
	})

	if err != nil {
		log.Error("Failed to verify relationships", zap.Error(err))
		os.Exit(1)
	}

	records = result.(neo4j.ResultWithContext)
	if records.Next(ctxWithTimeout) {
		record := records.Record()
		count := record.Values[0].(int64)
		log.Info("Final SENT_TO relationship count", zap.Int64("count", count))

		if count == 0 {
			log.Error("No SENT_TO relationships found after migration!", zap.Error(err))
			os.Exit(1)
		}
	}

	log.Info("Cleanup complete! Database now contains Wallet nodes with direct SENT_TO relationships including transaction details.")
}
