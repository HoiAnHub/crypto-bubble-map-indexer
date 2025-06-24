package database

import (
	"context"
	"fmt"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/domain/repository"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/zap"
)

// Neo4JTransactionRepository implements TransactionRepository interface
type Neo4JTransactionRepository struct {
	client *Neo4JClient
	logger *logger.Logger
}

// NewNeo4JTransactionRepository creates a new Neo4J transaction repository
func NewNeo4JTransactionRepository(client *Neo4JClient, logger *logger.Logger) repository.TransactionRepository {
	return &Neo4JTransactionRepository{
		client: client,
		logger: logger.WithComponent("neo4j-transaction-repo"),
	}
}

// CreateTransaction creates a new transaction node - deprecated, not creating Transaction nodes anymore
func (r *Neo4JTransactionRepository) CreateTransaction(ctx context.Context, tx *entity.TransactionNode) error {
	// No longer creating Transaction nodes, just return nil
	return nil
}

// CreateTransactionRelationship creates a direct relationship between wallets
func (r *Neo4JTransactionRepository) CreateTransactionRelationship(ctx context.Context, rel *entity.TransactionRelationship) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// First, ensure wallets exist
	createWalletsQuery := `
		MERGE (from:Wallet {address: $from_address})
		MERGE (to:Wallet {address: $to_address})
	`
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, createWalletsQuery, map[string]interface{}{
			"from_address": rel.FromAddress,
			"to_address":   rel.ToAddress,
		})
	})

	if err != nil {
		return fmt.Errorf("failed to ensure wallet nodes exist: %w", err)
	}

	// Then create or update the relationship
	query := `
		MATCH (from:Wallet {address: $from_address})
		MATCH (to:Wallet {address: $to_address})
		MERGE (from)-[r:SENT_TO]->(to)
		ON CREATE SET
			r.total_value = $value,
			r.tx_count = 1,
			r.first_tx = $timestamp,
			r.last_tx = $timestamp,
			r.tx_details = [{hash: $tx_hash, value: $value, timestamp: $timestamp}]
		ON MATCH SET
			r.total_value = toString(toFloat(r.total_value) + toFloat($value)),
			r.tx_count = CASE WHEN r.tx_count IS NULL THEN 1 ELSE r.tx_count + 1 END,
			r.last_tx = $timestamp,
			r.tx_details = CASE
				WHEN r.tx_details IS NULL THEN [{hash: $tx_hash, value: $value, timestamp: $timestamp}]
				ELSE r.tx_details + {hash: $tx_hash, value: $value, timestamp: $timestamp}
			END
	`

	params := map[string]interface{}{
		"from_address": rel.FromAddress,
		"to_address":   rel.ToAddress,
		"value":        rel.Value,
		"timestamp":    rel.Timestamp,
		"tx_hash":      rel.TxHash,
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, params)
	})

	if err != nil {
		return fmt.Errorf("failed to create transaction relationship: %w", err)
	}

	return nil
}

// GetTransaction retrieves a transaction by hash - deprecated, not querying Transaction nodes anymore
func (r *Neo4JTransactionRepository) GetTransaction(ctx context.Context, hash string) (*entity.TransactionNode, error) {
	return nil, fmt.Errorf("transaction nodes no longer supported")
}

// GetTransactionPath finds the path between two wallets
func (r *Neo4JTransactionRepository) GetTransactionPath(ctx context.Context, fromAddress, toAddress string, maxHops int) ([]*entity.TransactionNode, error) {
	// This now returns an empty result since we're not tracking Transaction nodes anymore
	return []*entity.TransactionNode{}, nil
}

// GetTransactionsByWallet retrieves transactions for a specific wallet - now returns empty result
func (r *Neo4JTransactionRepository) GetTransactionsByWallet(ctx context.Context, address string, limit int) ([]*entity.TransactionNode, error) {
	// This now returns an empty result since we're not tracking Transaction nodes anymore
	return []*entity.TransactionNode{}, nil
}

// GetTransactionsByTimeRange retrieves transactions within a time range - now returns empty result
func (r *Neo4JTransactionRepository) GetTransactionsByTimeRange(ctx context.Context, startTime, endTime string, limit int) ([]*entity.TransactionNode, error) {
	// This now returns an empty result since we're not tracking Transaction nodes anymore
	return []*entity.TransactionNode{}, nil
}

// BatchCreateTransactions creates multiple transactions in a batch - deprecated, not creating Transaction nodes anymore
func (r *Neo4JTransactionRepository) BatchCreateTransactions(ctx context.Context, transactions []*entity.TransactionNode) error {
	// No longer creating Transaction nodes, just return nil
	return nil
}

// BatchCreateRelationships creates multiple relationships in a batch
func (r *Neo4JTransactionRepository) BatchCreateRelationships(ctx context.Context, relationships []*entity.TransactionRelationship) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// First ensure all wallet nodes exist
	createWalletsQuery := `
		UNWIND $wallets AS wallet
		MERGE (w:Wallet {address: wallet})
	`

	walletsMap := make(map[string]bool)
	for _, rel := range relationships {
		walletsMap[rel.FromAddress] = true
		walletsMap[rel.ToAddress] = true
	}

	var walletsList []string
	for address := range walletsMap {
		walletsList = append(walletsList, address)
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, createWalletsQuery, map[string]interface{}{
			"wallets": walletsList,
		})
	})

	if err != nil {
		return fmt.Errorf("failed to ensure wallet nodes exist: %w", err)
	}

	// Then create or update the relationships
	query := `
		UNWIND $relationships as rel
		MATCH (from:Wallet {address: rel.from_address})
		MATCH (to:Wallet {address: rel.to_address})
		MERGE (from)-[r:SENT_TO]->(to)
		ON CREATE SET
			r.total_value = rel.value,
			r.tx_count = 1,
			r.first_tx = datetime(rel.timestamp),
			r.last_tx = datetime(rel.timestamp),
			r.tx_details = [{hash: rel.tx_hash, value: rel.value, timestamp: datetime(rel.timestamp)}]
		ON MATCH SET
			r.total_value = toString(toFloat(r.total_value) + toFloat(rel.value)),
			r.tx_count = CASE WHEN r.tx_count IS NULL THEN 1 ELSE r.tx_count + 1 END,
			r.last_tx = datetime(rel.timestamp),
			r.tx_details = CASE
				WHEN r.tx_details IS NULL THEN [{hash: rel.tx_hash, value: rel.value, timestamp: datetime(rel.timestamp)}]
				ELSE r.tx_details + {hash: rel.tx_hash, value: rel.value, timestamp: datetime(rel.timestamp)}
			END
	`

	var relData []map[string]interface{}
	for _, rel := range relationships {
		// Format the timestamp as ISO-8601 string for Neo4J
		timestampStr := rel.Timestamp.Format("2006-01-02T15:04:05.000Z")

		relData = append(relData, map[string]interface{}{
			"from_address": rel.FromAddress,
			"to_address":   rel.ToAddress,
			"value":        rel.Value,
			"timestamp":    timestampStr,
			"tx_hash":      rel.TxHash,
		})
	}

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"relationships": relData})
	})

	if err != nil {
		return fmt.Errorf("failed to batch create relationships: %w", err)
	}

	// Verify relationships were created
	verifyQuery := `
		MATCH ()-[r:SENT_TO]->()
		RETURN count(r) as relationship_count
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, verifyQuery, map[string]interface{}{})
	})

	if err != nil {
		r.logger.Warn("Failed to verify relationships", zap.Error(err))
		return nil
	}

	records := result.(neo4j.ResultWithContext)
	if records.Next(ctx) {
		count := records.Record().Values[0].(int64)
		r.logger.Info("Current SENT_TO relationship count", zap.Int64("count", count))
	}

	return nil
}
