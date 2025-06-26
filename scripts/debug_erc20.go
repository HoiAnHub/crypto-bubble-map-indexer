package main

import (
	"context"
	"fmt"
	"time"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/infrastructure/blockchain"
	"crypto-bubble-map-indexer/internal/infrastructure/config"
	"crypto-bubble-map-indexer/internal/infrastructure/database"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"go.uber.org/zap"
)

func main() {
	// Setup logger with debug level
	log, err := logger.NewLogger("debug")
	if err != nil {
		panic(err)
	}
	log = log.WithComponent("erc20-debug")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config", zap.Error(err))
	}

	// Create ERC20 decoder
	decoder := blockchain.NewERC20DecoderService(log)

	// Create test transactions
	testTransactions := createTestTransactions()

	log.Info("Testing ERC20 decoder with sample transactions",
		zap.Int("test_count", len(testTransactions)))

	ctx := context.Background()

	// Test each transaction
	for i, tx := range testTransactions {
		log.Info("Testing transaction",
			zap.Int("test_number", i+1),
			zap.String("tx_hash", tx.Hash),
			zap.String("description", tx.Network)) // Using network field for test description

		transfers, err := decoder.DecodeERC20Transfer(ctx, tx)
		if err != nil {
			log.Error("Failed to decode ERC20 transfer",
				zap.String("tx_hash", tx.Hash),
				zap.Error(err))
			continue
		}

		if len(transfers) == 0 {
			log.Warn("No ERC20 transfers found",
				zap.String("tx_hash", tx.Hash))
		} else {
			log.Info("Found ERC20 transfers",
				zap.String("tx_hash", tx.Hash),
				zap.Int("count", len(transfers)))

			for j, transfer := range transfers {
				log.Info("ERC20 Transfer",
					zap.Int("transfer_number", j+1),
					zap.String("contract", transfer.ContractAddress),
					zap.String("from", transfer.From),
					zap.String("to", transfer.To),
					zap.String("value", transfer.Value))
			}
		}
	}

	// Test database connection and insertion
	log.Info("Testing database connection and ERC20 insertion...")
	if err := testDatabaseInsertion(ctx, cfg, log); err != nil {
		log.Error("Database test failed", zap.Error(err))
	} else {
		log.Info("Database test completed successfully")
	}
}

func createTestTransactions() []*entity.Transaction {
	now := time.Now()

	return []*entity.Transaction{
		// Test 1: Simple ERC20 transfer
		{
			Hash:        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			From:        "0x742d35Cc6634C0532925a3b8D0c5d76C9C9f2A2D",
			To:          "0xA0b86a33E6b06B1C3aAe5E6A3E7b5c5d6fC8F4C0", // ERC20 contract
			Value:       "0",
			Data:        "0xa9059cbb000000000000000000000000742d35cc6634c0532925a3b8d0c5d76c9c9f2a2d0000000000000000000000000000000000000000000000000de0b6b3a7640000", // transfer function
			BlockNumber: "12345678",
			Timestamp:   now,
			Network:     "ERC20 transfer test",
		},
		// Test 2: transferFrom call
		{
			Hash:        "0x2234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			From:        "0x742d35Cc6634C0532925a3b8D0c5d76C9C9f2A2D",
			To:          "0xA0b86a33E6b06B1C3aAe5E6A3E7b5c5d6fC8F4C0", // ERC20 contract
			Value:       "0",
			Data:        "0x23b872dd000000000000000000000000742d35cc6634c0532925a3b8d0c5d76c9c9f2a2d000000000000000000000000123456789abcdef123456789abcdef123456789ab0000000000000000000000000000000000000000000000000de0b6b3a7640000", // transferFrom function
			BlockNumber: "12345679",
			Timestamp:   now,
			Network:     "ERC20 transferFrom test",
		},
		// Test 3: Contract interaction with ETH value
		{
			Hash:        "0x3234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			From:        "0x742d35Cc6634C0532925a3b8D0c5d76C9C9f2A2D",
			To:          "0xA0b86a33E6b06B1C3aAe5E6A3E7b5c5d6fC8F4C0", // Contract
			Value:       "1000000000000000000",                        // 1 ETH
			Data:        "0x1234abcd",                                 // Some contract call
			BlockNumber: "12345680",
			Timestamp:   now,
			Network:     "Contract interaction with ETH value test",
		},
		// Test 4: Simple ETH transfer (no contract data)
		{
			Hash:        "0x4234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			From:        "0x742d35Cc6634C0532925a3b8D0c5d76C9C9f2A2D",
			To:          "0x123456789abcdef123456789abcdef123456789ab",
			Value:       "1000000000000000000", // 1 ETH
			Data:        "",
			BlockNumber: "12345681",
			Timestamp:   now,
			Network:     "Simple ETH transfer test",
		},
		// Test 5: Unknown method signature
		{
			Hash:        "0x5234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			From:        "0x742d35Cc6634C0532925a3b8D0c5d76C9C9f2A2D",
			To:          "0xA0b86a33E6b06B1C3aAe5E6A3E7b5c5d6fC8F4C0",
			Value:       "0",
			Data:        "0x12345678000000000000000000000000742d35cc6634c0532925a3b8d0c5d76c9c9f2a2d",
			BlockNumber: "12345682",
			Timestamp:   now,
			Network:     "Unknown method signature test",
		},
	}
}

func testDatabaseInsertion(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	// Create Neo4j client
	neo4jClient := database.NewNeo4JClient(&cfg.Neo4J, log)

	// Connect to Neo4j
	err := neo4jClient.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Neo4j: %w", err)
	}
	defer neo4jClient.Close(ctx)

	// Create ERC20 repository
	erc20Repo := database.NewNeo4JERC20Repository(neo4jClient, log)

	// Test creating an ERC20 contract
	testContract := &entity.ERC20Contract{
		Address:   "0xA0b86a33E6b06B1C3aAe5E6A3E7b5c5d6fC8F4C0",
		Name:      "Test Token",
		Symbol:    "TEST",
		Decimals:  18,
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
		TotalTxs:  1,
		Network:   "ethereum",
	}

	log.Info("Creating test ERC20 contract", zap.String("address", testContract.Address))
	if err := erc20Repo.CreateOrUpdateERC20Contract(ctx, testContract); err != nil {
		return fmt.Errorf("failed to create test ERC20 contract: %w", err)
	}

	// Test creating an ERC20 transfer relationship
	testTransfer := &entity.ERC20TransferRelationship{
		FromAddress:     "0x742d35Cc6634C0532925a3b8D0c5d76C9C9f2A2D",
		ToAddress:       "0x123456789abcdef123456789abcdef123456789ab",
		ContractAddress: "0xA0b86a33E6b06B1C3aAe5E6A3E7b5c5d6fC8F4C0",
		Value:           "1000000000000000000",
		TxHash:          "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Timestamp:       time.Now(),
		Network:         "ethereum",
	}

	log.Info("Creating test ERC20 transfer relationship",
		zap.String("from", testTransfer.FromAddress),
		zap.String("to", testTransfer.ToAddress),
		zap.String("contract", testTransfer.ContractAddress))

	if err := erc20Repo.CreateERC20TransferRelationship(ctx, testTransfer); err != nil {
		return fmt.Errorf("failed to create test ERC20 transfer relationship: %w", err)
	}

	log.Info("Successfully created test data in database")
	return nil
}
