package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/infrastructure/config"
	"crypto-bubble-map-indexer/internal/infrastructure/database"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	fmt.Println("🔍 Testing TX_Details Approach for ERC20 Relationships")
	fmt.Println("=====================================================")

	// Initialize logger
	logger, err := logger.NewLogger("debug")
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize Neo4j client
	neo4jClient := database.NewNeo4JClient(&cfg.Neo4J, logger)
	ctx := context.Background()

	// Connect to Neo4j
	err = neo4jClient.Connect(ctx)
	if err != nil {
		log.Fatal("Failed to connect to Neo4j:", err)
	}
	defer neo4jClient.Close(ctx)

	// Initialize repository
	erc20Repo := database.NewNeo4JERC20Repository(neo4jClient, logger)

	// Test sample ERC20 relationships with tx_details
	testRelationships := []*entity.ERC20TransferRelationship{
		{
			FromAddress:      "0x1111111111111111111111111111111111111111",
			ToAddress:        "0x2222222222222222222222222222222222222222",
			ContractAddress:  "0xa0b86a33e6441e01e5a7f92c1c7b0d0c5eb38e16", // ERC20 token
			Value:            "1000000000000000000",
			TxHash:           "0xabc123",
			Timestamp:        time.Now().Add(-time.Hour),
			Network:          "ethereum",
			InteractionType:  entity.InteractionTransfer,
			MethodSignature:  "a9059cbb",
			TotalValue:       "1000000000000000000",
			TransactionCount: 1,
			FirstInteraction: time.Now().Add(-time.Hour),
			LastInteraction:  time.Now().Add(-time.Hour),
		},
		{
			FromAddress:      "0x1111111111111111111111111111111111111111",
			ToAddress:        "0x3333333333333333333333333333333333333333",
			ContractAddress:  "0xa0b86a33e6441e01e5a7f92c1c7b0d0c5eb38e16", // Same ERC20 token
			Value:            "2000000000000000000",
			TxHash:           "0xdef456",
			Timestamp:        time.Now().Add(-30 * time.Minute),
			Network:          "ethereum",
			InteractionType:  entity.InteractionTransfer,
			MethodSignature:  "a9059cbb",
			TotalValue:       "2000000000000000000",
			TransactionCount: 1,
			FirstInteraction: time.Now().Add(-30 * time.Minute),
			LastInteraction:  time.Now().Add(-30 * time.Minute),
		},
		{
			FromAddress:      "0x1111111111111111111111111111111111111111",
			ToAddress:        "0x7a250d5630b4cf539739df2c5dacb4c659f2488d", // Uniswap V2 Router
			ContractAddress:  "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
			Value:            "100000000000000000",
			TxHash:           "0x789ghi",
			Timestamp:        time.Now().Add(-15 * time.Minute),
			Network:          "ethereum",
			InteractionType:  entity.InteractionSwap,
			MethodSignature:  "7ff36ab5",
			TotalValue:       "100000000000000000",
			TransactionCount: 1,
			FirstInteraction: time.Now().Add(-15 * time.Minute),
			LastInteraction:  time.Now().Add(-15 * time.Minute),
		},
		{
			FromAddress:      "0x1111111111111111111111111111111111111111",
			ToAddress:        "0x4444444444444444444444444444444444444444",
			ContractAddress:  "0xa0b86a33e6441e01e5a7f92c1c7b0d0c5eb38e16", // Same token, another approval
			Value:            "115792089237316195423570985008687907853269984665640564039457584007913129639935",
			TxHash:           "0x111jkl",
			Timestamp:        time.Now().Add(-5 * time.Minute),
			Network:          "ethereum",
			InteractionType:  entity.InteractionApprove,
			MethodSignature:  "095ea7b3",
			TotalValue:       "115792089237316195423570985008687907853269984665640564039457584007913129639935",
			TransactionCount: 1,
			FirstInteraction: time.Now().Add(-5 * time.Minute),
			LastInteraction:  time.Now().Add(-5 * time.Minute),
		},
	}

	fmt.Printf("Creating %d test ERC20 relationships...\n", len(testRelationships))

	// Create test relationships
	err = erc20Repo.BatchCreateERC20TransferRelationships(ctx, testRelationships)
	if err != nil {
		log.Fatalf("Failed to create test relationships: %v", err)
	}

	fmt.Println("✅ Test relationships created successfully!")

	// Query to check the tx_details
	fmt.Println("\n🔍 Querying relationships with tx_details...")

	session := neo4jClient.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Check ERC20_TRANSFER relationships
	fmt.Println("\n--- ERC20_TRANSFER Relationships ---")
	checkRelationshipTxDetails(ctx, session, "ERC20_TRANSFER")

	// Check DEX_SWAP relationships
	fmt.Println("\n--- DEX_SWAP Relationships ---")
	checkRelationshipTxDetails(ctx, session, "DEX_SWAP")

	// Check ERC20_APPROVAL relationships
	fmt.Println("\n--- ERC20_APPROVAL Relationships ---")
	checkRelationshipTxDetails(ctx, session, "ERC20_APPROVAL")

	// Show overall summary
	fmt.Println("\n📊 Summary of All Relationships:")
	showRelationshipSummary(ctx, session)

	fmt.Println("\n✅ TX_Details approach test completed!")
	fmt.Println("\n📝 Key Benefits:")
	fmt.Println("   ✅ Each relationship stores detailed transaction history")
	fmt.Println("   ✅ Easy to trace back to original transactions")
	fmt.Println("   ✅ Aggregated data (total_value, tx_count) for quick analysis")
	fmt.Println("   ✅ Individual tx_details for deep investigation")
}

func checkRelationshipTxDetails(ctx context.Context, session neo4j.SessionWithContext, relType string) {
	query := fmt.Sprintf(`
		MATCH ()-[r:%s]->()
		RETURN r.total_value, r.tx_count, r.tx_details, r.interaction_type, r.first_tx, r.last_tx
		LIMIT 5
	`, relType)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, nil)
	})

	if err != nil {
		fmt.Printf("   ❌ Error querying %s: %v\n", relType, err)
		return
	}

	records := result.(neo4j.ResultWithContext)
	count := 0

	for records.Next(ctx) {
		count++
		record := records.Record()
		values := record.Values

		totalValue := values[0]
		txCount := values[1]
		txDetails := values[2]
		interactionType := values[3]
		firstTx := values[4]
		lastTx := values[5]

		fmt.Printf("   Relationship #%d:\n", count)
		fmt.Printf("     Total Value: %v\n", totalValue)
		fmt.Printf("     TX Count: %v\n", txCount)
		fmt.Printf("     Interaction Type: %v\n", interactionType)
		fmt.Printf("     First TX: %v\n", firstTx)
		fmt.Printf("     Last TX: %v\n", lastTx)
		fmt.Printf("     TX Details: %v\n", txDetails)

		// Parse tx_details if it's an array
		if txDetailsArray, ok := txDetails.([]interface{}); ok {
			fmt.Printf("     Individual Transactions:\n")
			for i, detail := range txDetailsArray {
				fmt.Printf("       [%d] %v\n", i+1, detail)
			}
		}
		fmt.Println()
	}

	if count == 0 {
		fmt.Printf("   ⚠️  No %s relationships found\n", relType)
	} else {
		fmt.Printf("   ✅ Found %d %s relationships with tx_details\n", count, relType)
	}
}

func showRelationshipSummary(ctx context.Context, session neo4j.SessionWithContext) {
	query := `
		CALL db.relationshipTypes() YIELD relationshipType
		WITH relationshipType
		WHERE relationshipType IN ['ERC20_TRANSFER', 'ERC20_APPROVAL', 'DEX_SWAP', 'DEFI_OPERATION', 'LIQUIDITY_OPERATION', 'MULTICALL_OPERATION', 'ETH_TRANSFER', 'CONTRACT_INTERACTION']
		CALL {
			WITH relationshipType
			MATCH ()-[r]->()
			WHERE type(r) = relationshipType
			RETURN relationshipType, count(r) as count
		}
		RETURN relationshipType, count
		ORDER BY count DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, nil)
	})

	if err != nil {
		fmt.Printf("   ❌ Error querying relationship summary: %v\n", err)
		return
	}

	records := result.(neo4j.ResultWithContext)

	for records.Next(ctx) {
		record := records.Record()
		values := record.Values

		relType := values[0]
		count := values[1]

		fmt.Printf("   %s: %v relationships\n", relType, count)
	}
}
