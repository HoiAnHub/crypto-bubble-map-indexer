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

// Neo4JWalletRepository implements WalletRepository interface
type Neo4JWalletRepository struct {
	client *Neo4JClient
	logger *logger.Logger
}

// NewNeo4JWalletRepository creates a new Neo4J wallet repository
func NewNeo4JWalletRepository(client *Neo4JClient, logger *logger.Logger) repository.WalletRepository {
	return &Neo4JWalletRepository{
		client: client,
		logger: logger.WithComponent("neo4j-wallet-repo"),
	}
}

// CreateOrUpdateWallet creates a new wallet or updates existing one
func (r *Neo4JWalletRepository) CreateOrUpdateWallet(ctx context.Context, wallet *entity.Wallet) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (w:Wallet {address: $address})
		ON CREATE SET
			w.first_seen = datetime($first_seen),
			w.last_seen = datetime($last_seen),
			w.total_transactions = $total_transactions,
			w.total_sent = $total_sent,
			w.total_received = $total_received,
			w.network = $network
		ON MATCH SET
			w.last_seen = datetime($last_seen),
			w.total_transactions = $total_transactions,
			w.total_sent = $total_sent,
			w.total_received = $total_received
	`

	// Format the timestamp as ISO-8601 string for Neo4J
	firstSeenStr := wallet.FirstSeen.Format("2006-01-02T15:04:05.000Z")
	lastSeenStr := wallet.LastSeen.Format("2006-01-02T15:04:05.000Z")

	params := map[string]interface{}{
		"address":            wallet.Address,
		"first_seen":         firstSeenStr,
		"last_seen":          lastSeenStr,
		"total_transactions": wallet.TotalTransactions,
		"total_sent":         wallet.TotalSent,
		"total_received":     wallet.TotalReceived,
		"network":            wallet.Network,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, params)
	})

	if err != nil {
		return fmt.Errorf("failed to create/update wallet: %w", err)
	}

	return nil
}

// GetWallet retrieves a wallet by address
func (r *Neo4JWalletRepository) GetWallet(ctx context.Context, address string) (*entity.Wallet, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})
		RETURN w.address, w.first_seen, w.last_seen, w.total_transactions, w.total_sent, w.total_received, w.network
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"address": address})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	records := result.(neo4j.ResultWithContext)
	if !records.Next(ctx) {
		return nil, fmt.Errorf("wallet not found: %s", address)
	}

	record := records.Record()
	values := record.Values

	wallet := &entity.Wallet{
		Address:           values[0].(string),
		FirstSeen:         values[1].(time.Time),
		LastSeen:          values[2].(time.Time),
		TotalTransactions: values[3].(int64),
		TotalSent:         values[4].(string),
		TotalReceived:     values[5].(string),
		Network:           values[6].(string),
	}

	return wallet, nil
}

// GetWalletStats retrieves statistics for a wallet
func (r *Neo4JWalletRepository) GetWalletStats(ctx context.Context, address string) (*entity.WalletStats, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})
		OPTIONAL MATCH (w)-[:SENT_TO]->(other:Wallet)
		WITH w, count(other) as outgoing
		OPTIONAL MATCH (other2:Wallet)-[:SENT_TO]->(w)
		WITH w, outgoing, count(other2) as incoming
		OPTIONAL MATCH (w)-[r:SENT_TO]->()
		WITH w, outgoing, incoming, sum(toFloat(r.value)) as total_volume
		OPTIONAL MATCH (w)-[:SENT_TO|RECEIVED_FROM]->()
		RETURN w.address, incoming, outgoing, total_volume, count(*) as tx_count
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"address": address})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get wallet stats: %w", err)
	}

	records := result.(neo4j.ResultWithContext)
	if !records.Next(ctx) {
		return nil, fmt.Errorf("wallet not found: %s", address)
	}

	record := records.Record()
	values := record.Values

	stats := &entity.WalletStats{
		Address:             values[0].(string),
		IncomingConnections: values[1].(int64),
		OutgoingConnections: values[2].(int64),
		TotalVolume:         fmt.Sprintf("%.0f", values[3].(float64)),
		TransactionCount:    values[4].(int64),
	}

	return stats, nil
}

// GetWalletConnections retrieves connections for a wallet
func (r *Neo4JWalletRepository) GetWalletConnections(ctx context.Context, address string, limit int) ([]*entity.WalletConnection, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})-[r:SENT_TO]->(other:Wallet)
		WITH w, other, sum(toFloat(r.value)) as total_value, count(r) as tx_count, min(r.timestamp) as first_tx, max(r.timestamp) as last_tx
		RETURN w.address, other.address, total_value, tx_count, first_tx, last_tx
		ORDER BY total_value DESC
		LIMIT $limit
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address": address,
			"limit":   limit,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get wallet connections: %w", err)
	}

	var connections []*entity.WalletConnection
	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		connection := &entity.WalletConnection{
			FromAddress: values[0].(string),
			ToAddress:   values[1].(string),
			TotalValue:  fmt.Sprintf("%.0f", values[2].(float64)),
			TxCount:     values[3].(int64),
			FirstTx:     values[4].(time.Time),
			LastTx:      values[5].(time.Time),
		}
		connections = append(connections, connection)
	}

	return connections, nil
}

// FindConnectedWallets finds wallets connected to a given wallet within specified hops
func (r *Neo4JWalletRepository) FindConnectedWallets(ctx context.Context, address string, maxHops int) ([]*entity.Wallet, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH path = (w:Wallet {address: $address})-[*1..$max_hops]-(connected:Wallet)
		WHERE connected.address <> $address
		RETURN DISTINCT connected.address, connected.first_seen, connected.last_seen, connected.total_transactions, connected.total_sent, connected.total_received, connected.network
		LIMIT 100
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address":  address,
			"max_hops": maxHops,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find connected wallets: %w", err)
	}

	var wallets []*entity.Wallet
	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		wallet := &entity.Wallet{
			Address:           values[0].(string),
			FirstSeen:         values[1].(time.Time),
			LastSeen:          values[2].(time.Time),
			TotalTransactions: values[3].(int64),
			TotalSent:         values[4].(string),
			TotalReceived:     values[5].(string),
			Network:           values[6].(string),
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

// GetTopWallets retrieves top wallets by transaction count
func (r *Neo4JWalletRepository) GetTopWallets(ctx context.Context, limit int) ([]*entity.Wallet, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet)
		RETURN w.address, w.first_seen, w.last_seen, w.total_transactions, w.total_sent, w.total_received, w.network
		ORDER BY w.total_transactions DESC
		LIMIT $limit
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"limit": limit})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get top wallets: %w", err)
	}

	var wallets []*entity.Wallet
	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		wallet := &entity.Wallet{
			Address:           values[0].(string),
			FirstSeen:         values[1].(time.Time),
			LastSeen:          values[2].(time.Time),
			TotalTransactions: values[3].(int64),
			TotalSent:         values[4].(string),
			TotalReceived:     values[5].(string),
			Network:           values[6].(string),
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

// GetBubbleWallets retrieves wallets that form bubbles (high connectivity)
func (r *Neo4JWalletRepository) GetBubbleWallets(ctx context.Context, minConnections int, limit int) ([]*entity.Wallet, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet)-[:SENT_TO]->(other:Wallet)
		WITH w, count(other) as connections
		WHERE connections >= $min_connections
		RETURN w.address, w.first_seen, w.last_seen, w.total_transactions, w.total_sent, w.total_received, w.network, connections
		ORDER BY connections DESC
		LIMIT $limit
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"min_connections": minConnections,
			"limit":           limit,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get bubble wallets: %w", err)
	}

	var wallets []*entity.Wallet
	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		wallet := &entity.Wallet{
			Address:           values[0].(string),
			FirstSeen:         values[1].(time.Time),
			LastSeen:          values[2].(time.Time),
			TotalTransactions: values[3].(int64),
			TotalSent:         values[4].(string),
			TotalReceived:     values[5].(string),
			Network:           values[6].(string),
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}
