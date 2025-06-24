package database

import (
	"context"
	"fmt"
	"time"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/domain/repository"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
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

// CreateTransaction creates a new transaction node
func (r *Neo4JTransactionRepository) CreateTransaction(ctx context.Context, tx *entity.TransactionNode) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (t:Transaction {hash: $hash})
		ON CREATE SET
			t.block_number = $block_number,
			t.value = $value,
			t.gas_used = $gas_used,
			t.gas_price = $gas_price,
			t.timestamp = $timestamp,
			t.network = $network
	`

	params := map[string]interface{}{
		"hash":         tx.Hash,
		"block_number": tx.BlockNumber,
		"value":        tx.Value,
		"gas_used":     tx.GasUsed,
		"gas_price":    tx.GasPrice,
		"timestamp":    tx.Timestamp,
		"network":      tx.Network,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, params)
	})

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// CreateTransactionRelationship creates a relationship between wallets via transaction
func (r *Neo4JTransactionRepository) CreateTransactionRelationship(ctx context.Context, rel *entity.TransactionRelationship) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (from:Wallet {address: $from_address})
		MATCH (to:Wallet {address: $to_address})
		MATCH (tx:Transaction {hash: $tx_hash})
		MERGE (from)-[r:SENT_TO {tx_hash: $tx_hash}]->(to)
		SET r.value = $value,
			r.gas_price = $gas_price,
			r.timestamp = $timestamp
	`

	params := map[string]interface{}{
		"from_address": rel.FromAddress,
		"to_address":   rel.ToAddress,
		"tx_hash":      rel.TxHash,
		"value":        rel.Value,
		"gas_price":    rel.GasPrice,
		"timestamp":    rel.Timestamp,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, params)
	})

	if err != nil {
		return fmt.Errorf("failed to create transaction relationship: %w", err)
	}

	return nil
}

// GetTransaction retrieves a transaction by hash
func (r *Neo4JTransactionRepository) GetTransaction(ctx context.Context, hash string) (*entity.TransactionNode, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (t:Transaction {hash: $hash})
		RETURN t.hash, t.block_number, t.value, t.gas_used, t.gas_price, t.timestamp, t.network
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"hash": hash})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	records := result.(neo4j.ResultWithContext)
	if !records.Next(ctx) {
		return nil, fmt.Errorf("transaction not found: %s", hash)
	}

	record := records.Record()
	values := record.Values

	transaction := &entity.TransactionNode{
		Hash:        values[0].(string),
		BlockNumber: values[1].(string),
		Value:       values[2].(string),
		GasUsed:     values[3].(string),
		GasPrice:    values[4].(string),
		Timestamp:   values[5].(time.Time),
		Network:     values[6].(string),
	}

	return transaction, nil
}

// GetTransactionPath finds the path between two wallets through transactions
func (r *Neo4JTransactionRepository) GetTransactionPath(ctx context.Context, fromAddress, toAddress string, maxHops int) ([]*entity.TransactionNode, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH path = (from:Wallet {address: $from_address})-[*1..$max_hops]-(to:Wallet {address: $to_address})
		UNWIND relationships(path) as rel
		MATCH (tx:Transaction {hash: rel.tx_hash})
		RETURN DISTINCT tx.hash, tx.block_number, tx.value, tx.gas_used, tx.gas_price, tx.timestamp, tx.network
		ORDER BY tx.timestamp
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"from_address": fromAddress,
			"to_address":   toAddress,
			"max_hops":     maxHops,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction path: %w", err)
	}

	var transactions []*entity.TransactionNode
	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		transaction := &entity.TransactionNode{
			Hash:        values[0].(string),
			BlockNumber: values[1].(string),
			Value:       values[2].(string),
			GasUsed:     values[3].(string),
			GasPrice:    values[4].(string),
			Timestamp:   values[5].(time.Time),
			Network:     values[6].(string),
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// GetTransactionsByWallet retrieves transactions for a specific wallet
func (r *Neo4JTransactionRepository) GetTransactionsByWallet(ctx context.Context, address string, limit int) ([]*entity.TransactionNode, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})-[r:SENT_TO]->()
		MATCH (tx:Transaction {hash: r.tx_hash})
		RETURN tx.hash, tx.block_number, tx.value, tx.gas_used, tx.gas_price, tx.timestamp, tx.network
		ORDER BY tx.timestamp DESC
		LIMIT $limit
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address": address,
			"limit":   limit,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by wallet: %w", err)
	}

	var transactions []*entity.TransactionNode
	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		transaction := &entity.TransactionNode{
			Hash:        values[0].(string),
			BlockNumber: values[1].(string),
			Value:       values[2].(string),
			GasUsed:     values[3].(string),
			GasPrice:    values[4].(string),
			Timestamp:   values[5].(time.Time),
			Network:     values[6].(string),
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// GetTransactionsByTimeRange retrieves transactions within a time range
func (r *Neo4JTransactionRepository) GetTransactionsByTimeRange(ctx context.Context, startTime, endTime string, limit int) ([]*entity.TransactionNode, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (t:Transaction)
		WHERE t.timestamp >= datetime($start_time) AND t.timestamp <= datetime($end_time)
		RETURN t.hash, t.block_number, t.value, t.gas_used, t.gas_price, t.timestamp, t.network
		ORDER BY t.timestamp DESC
		LIMIT $limit
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"start_time": startTime,
			"end_time":   endTime,
			"limit":      limit,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by time range: %w", err)
	}

	var transactions []*entity.TransactionNode
	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		transaction := &entity.TransactionNode{
			Hash:        values[0].(string),
			BlockNumber: values[1].(string),
			Value:       values[2].(string),
			GasUsed:     values[3].(string),
			GasPrice:    values[4].(string),
			Timestamp:   values[5].(time.Time),
			Network:     values[6].(string),
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// BatchCreateTransactions creates multiple transactions in a batch
func (r *Neo4JTransactionRepository) BatchCreateTransactions(ctx context.Context, transactions []*entity.TransactionNode) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		UNWIND $transactions as tx
		MERGE (t:Transaction {hash: tx.hash})
		ON CREATE SET
			t.block_number = tx.block_number,
			t.value = tx.value,
			t.gas_used = tx.gas_used,
			t.gas_price = tx.gas_price,
			t.timestamp = datetime(tx.timestamp),
			t.network = tx.network
	`

	var txData []map[string]interface{}
	for _, tx := range transactions {
		// Format the timestamp as ISO-8601 string for Neo4J
		timestampStr := tx.Timestamp.Format("2006-01-02T15:04:05.000Z")

		txData = append(txData, map[string]interface{}{
			"hash":         tx.Hash,
			"block_number": tx.BlockNumber,
			"value":        tx.Value,
			"gas_used":     tx.GasUsed,
			"gas_price":    tx.GasPrice,
			"timestamp":    timestampStr,
			"network":      tx.Network,
		})
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"transactions": txData})
	})

	if err != nil {
		return fmt.Errorf("failed to batch create transactions: %w", err)
	}

	return nil
}

// BatchCreateRelationships creates multiple relationships in a batch
func (r *Neo4JTransactionRepository) BatchCreateRelationships(ctx context.Context, relationships []*entity.TransactionRelationship) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		UNWIND $relationships as rel
		MATCH (from:Wallet {address: rel.from_address})
		MATCH (to:Wallet {address: rel.to_address})
		MATCH (tx:Transaction {hash: rel.tx_hash})
		MERGE (from)-[r:SENT_TO {tx_hash: rel.tx_hash}]->(to)
		SET r.value = rel.value,
			r.gas_price = rel.gas_price,
			r.timestamp = datetime(rel.timestamp)
	`

	var relData []map[string]interface{}
	for _, rel := range relationships {
		// Format the timestamp as ISO-8601 string for Neo4J
		timestampStr := rel.Timestamp.Format("2006-01-02T15:04:05.000Z")

		relData = append(relData, map[string]interface{}{
			"from_address": rel.FromAddress,
			"to_address":   rel.ToAddress,
			"tx_hash":      rel.TxHash,
			"value":        rel.Value,
			"gas_price":    rel.GasPrice,
			"timestamp":    timestampStr,
		})
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"relationships": relData})
	})

	if err != nil {
		return fmt.Errorf("failed to batch create relationships: %w", err)
	}

	return nil
}
