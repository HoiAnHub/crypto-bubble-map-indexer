package database

import (
	"context"
	"fmt"
	"time"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/domain/repository"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/zap"
)

// Neo4JERC20Repository implements ERC20Repository using Neo4J
type Neo4JERC20Repository struct {
	client *Neo4JClient
	logger *logger.Logger
}

// NewNeo4JERC20Repository creates a new Neo4J ERC20 repository
func NewNeo4JERC20Repository(client *Neo4JClient, logger *logger.Logger) repository.ERC20Repository {
	return &Neo4JERC20Repository{
		client: client,
		logger: logger.WithComponent("neo4j-erc20-repository"),
	}
}

// CreateOrUpdateERC20Contract creates or updates an ERC20 contract
func (r *Neo4JERC20Repository) CreateOrUpdateERC20Contract(ctx context.Context, contract *entity.ERC20Contract) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (c:ERC20Contract {address: $address})
		ON CREATE SET
			c.name = $name,
			c.symbol = $symbol,
			c.decimals = $decimals,
			c.first_seen = $first_seen,
			c.last_seen = $last_seen,
			c.total_txs = $total_txs,
			c.network = $network
		ON MATCH SET
			c.last_seen = $last_seen,
			c.total_txs = c.total_txs + 1
	`

	parameters := map[string]interface{}{
		"address":    contract.Address,
		"name":       contract.Name,
		"symbol":     contract.Symbol,
		"decimals":   contract.Decimals,
		"first_seen": contract.FirstSeen.Format("2006-01-02T15:04:05.000Z"),
		"last_seen":  contract.LastSeen.Format("2006-01-02T15:04:05.000Z"),
		"total_txs":  contract.TotalTxs,
		"network":    contract.Network,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, parameters)
	})

	if err != nil {
		r.logger.Error("Failed to create/update ERC20 contract",
			zap.String("address", contract.Address),
			zap.Error(err))
		return fmt.Errorf("failed to create/update ERC20 contract: %w", err)
	}

	return nil
}

// CreateERC20TransferRelationship creates a transfer relationship between wallets
func (r *Neo4JERC20Repository) CreateERC20TransferRelationship(ctx context.Context, transfer *entity.ERC20TransferRelationship) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (from:Wallet {address: $from_address})
		MERGE (to:Wallet {address: $to_address})
		MERGE (contract:ERC20Contract {address: $contract_address})
		CREATE (from)-[:ERC20_TRANSFER {
			value: $value,
			tx_hash: $tx_hash,
			timestamp: $timestamp,
			network: $network,
			contract_address: $contract_address
		}]->(to)
		CREATE (from)-[:INTERACTED_WITH]->(contract)
		CREATE (to)-[:INTERACTED_WITH]->(contract)
	`

	parameters := map[string]interface{}{
		"from_address":     transfer.FromAddress,
		"to_address":       transfer.ToAddress,
		"contract_address": transfer.ContractAddress,
		"value":            transfer.Value,
		"tx_hash":          transfer.TxHash,
		"timestamp":        transfer.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		"network":          transfer.Network,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, parameters)
	})

	if err != nil {
		r.logger.Error("Failed to create ERC20 transfer relationship",
			zap.String("from", transfer.FromAddress),
			zap.String("to", transfer.ToAddress),
			zap.String("contract", transfer.ContractAddress),
			zap.Error(err))
		return fmt.Errorf("failed to create ERC20 transfer relationship: %w", err)
	}

	return nil
}

// BatchCreateERC20TransferRelationships creates multiple ERC20 transfer relationships in a batch
func (r *Neo4JERC20Repository) BatchCreateERC20TransferRelationships(ctx context.Context, relationships []*entity.ERC20TransferRelationship) error {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Group relationships by type for optimal processing
	relationshipGroups := make(map[string][]*entity.ERC20TransferRelationship)
	for _, rel := range relationships {
		relType := rel.InteractionType.GetRelationshipType()
		relationshipGroups[relType] = append(relationshipGroups[relType], rel)
	}

	r.logger.Info("Creating ERC20 relationships in batches",
		zap.Int("total_relationships", len(relationships)),
		zap.Int("relationship_types", len(relationshipGroups)))

	// Process each relationship type separately
	for relType, rels := range relationshipGroups {
		if err := r.batchCreateRelationshipsByType(ctx, session, relType, rels); err != nil {
			return fmt.Errorf("failed to create %s relationships: %w", relType, err)
		}
		r.logger.Debug("Created relationships by type",
			zap.String("relationship_type", relType),
			zap.Int("count", len(rels)))
	}

	return nil
}

// batchCreateRelationshipsByType creates relationships of a specific type
func (r *Neo4JERC20Repository) batchCreateRelationshipsByType(ctx context.Context, session neo4j.SessionWithContext, relType string, relationships []*entity.ERC20TransferRelationship) error {
	// Different queries for different relationship types
	var query string

	switch relType {
	case "ERC20_TRANSFER":
		// For transfers, create relationship between wallets
		query = `
			UNWIND $relationships as rel
			MATCH (from:Wallet {address: rel.from_address})
			MATCH (to:Wallet {address: rel.to_address})
			MATCH (contract:ERC20Contract {address: rel.contract_address})
			MERGE (from)-[r:ERC20_TRANSFER {contract_address: rel.contract_address}]->(to)
			ON CREATE SET
				r.total_value = rel.value,
				r.tx_count = 1,
				r.first_tx = datetime(rel.timestamp),
				r.last_tx = datetime(rel.timestamp),
				r.interaction_type = rel.interaction_type,
				r.contract_address = rel.contract_address,
				r.network = rel.network
			ON MATCH SET
				r.total_value = toString(toFloat(r.total_value) + toFloat(rel.value)),
				r.tx_count = r.tx_count + 1,
				r.last_tx = datetime(rel.timestamp)
		`

	case "ERC20_APPROVAL":
		// For approvals, create relationship from wallet to contract/spender
		query = `
			UNWIND $relationships as rel
			MATCH (from:Wallet {address: rel.from_address})
			MATCH (to:Wallet {address: rel.to_address})
			MATCH (contract:ERC20Contract {address: rel.contract_address})
			MERGE (from)-[r:ERC20_APPROVAL {contract_address: rel.contract_address, spender: rel.to_address}]->(contract)
			ON CREATE SET
				r.total_value = rel.value,
				r.tx_count = 1,
				r.first_tx = datetime(rel.timestamp),
				r.last_tx = datetime(rel.timestamp),
				r.interaction_type = rel.interaction_type,
				r.spender = rel.to_address,
				r.network = rel.network
			ON MATCH SET
				r.total_value = rel.value,
				r.tx_count = r.tx_count + 1,
				r.last_tx = datetime(rel.timestamp)
		`

	case "DEX_SWAP":
		// For swaps, create relationship from wallet to DEX contract
		query = `
			UNWIND $relationships as rel
			MATCH (from:Wallet {address: rel.from_address})
			MATCH (contract:ERC20Contract {address: rel.contract_address})
			MERGE (from)-[r:DEX_SWAP {contract_address: rel.contract_address}]->(contract)
			ON CREATE SET
				r.total_value = rel.value,
				r.tx_count = 1,
				r.first_tx = datetime(rel.timestamp),
				r.last_tx = datetime(rel.timestamp),
				r.interaction_type = rel.interaction_type,
				r.network = rel.network
			ON MATCH SET
				r.total_value = toString(toFloat(r.total_value) + toFloat(rel.value)),
				r.tx_count = r.tx_count + 1,
				r.last_tx = datetime(rel.timestamp)
		`

	case "LIQUIDITY_OPERATION":
		// For liquidity operations
		query = `
			UNWIND $relationships as rel
			MATCH (from:Wallet {address: rel.from_address})
			MATCH (contract:ERC20Contract {address: rel.contract_address})
			MERGE (from)-[r:LIQUIDITY_OPERATION {contract_address: rel.contract_address}]->(contract)
			ON CREATE SET
				r.total_value = rel.value,
				r.tx_count = 1,
				r.first_tx = datetime(rel.timestamp),
				r.last_tx = datetime(rel.timestamp),
				r.interaction_type = rel.interaction_type,
				r.network = rel.network
			ON MATCH SET
				r.total_value = toString(toFloat(r.total_value) + toFloat(rel.value)),
				r.tx_count = r.tx_count + 1,
				r.last_tx = datetime(rel.timestamp)
		`

	case "DEFI_OPERATION":
		// For DeFi operations (deposit/withdraw)
		query = `
			UNWIND $relationships as rel
			MATCH (from:Wallet {address: rel.from_address})
			MATCH (contract:ERC20Contract {address: rel.contract_address})
			MERGE (from)-[r:DEFI_OPERATION {contract_address: rel.contract_address}]->(contract)
			ON CREATE SET
				r.total_value = rel.value,
				r.tx_count = 1,
				r.first_tx = datetime(rel.timestamp),
				r.last_tx = datetime(rel.timestamp),
				r.interaction_type = rel.interaction_type,
				r.network = rel.network
			ON MATCH SET
				r.total_value = toString(toFloat(r.total_value) + toFloat(rel.value)),
				r.tx_count = r.tx_count + 1,
				r.last_tx = datetime(rel.timestamp)
		`

	case "ETH_TRANSFER":
		// For ETH transfers, create simple relationship
		query = `
			UNWIND $relationships as rel
			MATCH (from:Wallet {address: rel.from_address})
			MATCH (to:Wallet {address: rel.to_address})
			MERGE (from)-[r:ETH_TRANSFER]->(to)
			ON CREATE SET
				r.total_value = rel.value,
				r.tx_count = 1,
				r.first_tx = datetime(rel.timestamp),
				r.last_tx = datetime(rel.timestamp),
				r.interaction_type = rel.interaction_type,
				r.network = rel.network
			ON MATCH SET
				r.total_value = toString(toFloat(r.total_value) + toFloat(rel.value)),
				r.tx_count = r.tx_count + 1,
				r.last_tx = datetime(rel.timestamp)
		`

	default:
		// For unknown contract interactions
		query = `
			UNWIND $relationships as rel
			MATCH (from:Wallet {address: rel.from_address})
			MATCH (contract:ERC20Contract {address: rel.contract_address})
			MERGE (from)-[r:CONTRACT_INTERACTION {contract_address: rel.contract_address}]->(contract)
			ON CREATE SET
				r.total_value = rel.value,
				r.tx_count = 1,
				r.first_tx = datetime(rel.timestamp),
				r.last_tx = datetime(rel.timestamp),
				r.interaction_type = rel.interaction_type,
				r.network = rel.network
			ON MATCH SET
				r.total_value = toString(toFloat(r.total_value) + toFloat(rel.value)),
				r.tx_count = r.tx_count + 1,
				r.last_tx = datetime(rel.timestamp)
		`
	}

	// Prepare relationship data
	var relData []map[string]interface{}
	for _, rel := range relationships {
		timestampStr := rel.Timestamp.Format("2006-01-02T15:04:05.000Z")

		relData = append(relData, map[string]interface{}{
			"from_address":     rel.FromAddress,
			"to_address":       rel.ToAddress,
			"contract_address": rel.ContractAddress,
			"value":            rel.Value,
			"timestamp":        timestampStr,
			"interaction_type": string(rel.InteractionType),
			"network":          rel.Network,
		})
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"relationships": relData})
	})

	if err != nil {
		return fmt.Errorf("failed to execute batch create for %s: %w", relType, err)
	}

	return nil
}

// GetERC20Contract retrieves an ERC20 contract by address
func (r *Neo4JERC20Repository) GetERC20Contract(ctx context.Context, address string) (*entity.ERC20Contract, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (c:ERC20Contract {address: $address})
		RETURN c.address, c.name, c.symbol, c.decimals, c.first_seen, c.last_seen, c.total_txs, c.network
	`

	parameters := map[string]interface{}{
		"address": address,
	}

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, parameters)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get ERC20 contract: %w", err)
	}

	records := result.(neo4j.ResultWithContext)
	if !records.Next(ctx) {
		return nil, fmt.Errorf("ERC20 contract not found: %s", address)
	}

	record := records.Record()
	values := record.Values

	contract := &entity.ERC20Contract{
		Address:   values[0].(string),
		Name:      values[1].(string),
		Symbol:    values[2].(string),
		Decimals:  int(values[3].(int64)),
		FirstSeen: values[4].(time.Time),
		LastSeen:  values[5].(time.Time),
		TotalTxs:  values[6].(int64),
		Network:   values[7].(string),
	}

	return contract, nil
}

// GetERC20TransfersBetweenWallets retrieves ERC20 transfers between two wallets
func (r *Neo4JERC20Repository) GetERC20TransfersBetweenWallets(ctx context.Context, fromAddress, toAddress string, limit int) ([]*entity.ERC20Transfer, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (from:Wallet {address: $from_address})-[r:ERC20_TRANSFER]->(to:Wallet {address: $to_address})
		RETURN r.contract_address, r.value, r.tx_hash, r.timestamp, r.network, from.address, to.address, ''
		ORDER BY r.timestamp DESC
		LIMIT $limit
	`

	parameters := map[string]interface{}{
		"from_address": fromAddress,
		"to_address":   toAddress,
		"limit":        limit,
	}

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, parameters)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get ERC20 transfers between wallets: %w", err)
	}

	var transfers []*entity.ERC20Transfer
	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		transfer := &entity.ERC20Transfer{
			ContractAddress: values[0].(string),
			Value:           values[1].(string),
			TxHash:          values[2].(string),
			Timestamp:       values[3].(time.Time),
			Network:         values[4].(string),
			From:            values[5].(string),
			To:              values[6].(string),
			BlockNumber:     values[7].(string),
		}
		transfers = append(transfers, transfer)
	}

	return transfers, nil
}

// GetERC20TransfersForWallet retrieves all ERC20 transfers for a wallet
func (r *Neo4JERC20Repository) GetERC20TransfersForWallet(ctx context.Context, address string, limit int) ([]*entity.ERC20Transfer, error) {
	session := r.client.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})
		MATCH (w)-[r:ERC20_TRANSFER]-(other:Wallet)
		RETURN r.contract_address, r.value, r.tx_hash, r.timestamp, r.network,
			   startNode(r).address, endNode(r).address, ''
		ORDER BY r.timestamp DESC
		LIMIT $limit
	`

	parameters := map[string]interface{}{
		"address": address,
		"limit":   limit,
	}

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, parameters)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get ERC20 transfers for wallet: %w", err)
	}

	var transfers []*entity.ERC20Transfer
	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		transfer := &entity.ERC20Transfer{
			ContractAddress: values[0].(string),
			Value:           values[1].(string),
			TxHash:          values[2].(string),
			Timestamp:       values[3].(time.Time),
			Network:         values[4].(string),
			From:            values[5].(string),
			To:              values[6].(string),
			BlockNumber:     values[7].(string),
		}
		transfers = append(transfers, transfer)
	}

	return transfers, nil
}
