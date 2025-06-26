package main

import (
	"context"
	"fmt"
	"log"

	"crypto-bubble-map-indexer/internal/infrastructure/config"
	"crypto-bubble-map-indexer/internal/infrastructure/database"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	fmt.Println("🔍 Verifying TX_Details Implementation")
	fmt.Println("=====================================")

	// Initialize logger
	logger, err := logger.NewLogger("info")
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

	session := neo4jClient.GetDriver().NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Query 1: Check all relationship types
	fmt.Println("1. All relationship types in database:")
	query1 := "CALL db.relationshipTypes() YIELD relationshipType RETURN relationshipType ORDER BY relationshipType"
	result1, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query1, nil)
	})
	if err == nil {
		records1 := result1.(neo4j.ResultWithContext)
		for records1.Next(ctx) {
			relType := records1.Record().Values[0]
			fmt.Printf("   - %v\n", relType)
		}
	}

	// Query 2: Count relationships by type
	fmt.Println("\n2. Relationship counts:")
	relationshipTypes := []string{"ERC20_TRANSFER", "ERC20_APPROVAL", "DEX_SWAP", "DEFI_OPERATION", "SENT_TO", "CONTRACT_INTERACTION"}

	for _, relType := range relationshipTypes {
		query := fmt.Sprintf("MATCH ()-[r:%s]->() RETURN count(r) as count", relType)
		result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			return tx.Run(ctx, query, nil)
		})
		if err == nil {
			records := result.(neo4j.ResultWithContext)
			if records.Next(ctx) {
				count := records.Record().Values[0]
				fmt.Printf("   %s: %v\n", relType, count)
			}
		}
	}

	// Query 3: Sample relationships with tx_details
	fmt.Println("\n3. Sample relationships with tx_details:")
	query3 := `
		MATCH ()-[r]->()
		WHERE exists(r.tx_details)
		RETURN type(r) as rel_type, r.tx_details[0] as first_tx_detail, size(r.tx_details) as detail_count
		LIMIT 5
	`
	result3, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query3, nil)
	})
	if err == nil {
		records3 := result3.(neo4j.ResultWithContext)
		for records3.Next(ctx) {
			record := records3.Record()
			relType := record.Values[0]
			firstDetail := record.Values[1]
			detailCount := record.Values[2]
			fmt.Printf("   %v: %v details, sample: %v\n", relType, detailCount, firstDetail)
		}
	}

	// Query 4: ERC20_TRANSFER specific check
	fmt.Println("\n4. ERC20_TRANSFER details:")
	query4 := `
		MATCH ()-[r:ERC20_TRANSFER]->()
		RETURN r.total_value, r.tx_count, r.interaction_type, r.tx_details
		LIMIT 3
	`
	result4, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query4, nil)
	})
	if err == nil {
		records4 := result4.(neo4j.ResultWithContext)
		count := 0
		for records4.Next(ctx) {
			count++
			record := records4.Record()
			totalValue := record.Values[0]
			txCount := record.Values[1]
			interactionType := record.Values[2]
			txDetails := record.Values[3]

			fmt.Printf("   Relationship #%d:\n", count)
			fmt.Printf("     Total Value: %v\n", totalValue)
			fmt.Printf("     TX Count: %v\n", txCount)
			fmt.Printf("     Interaction Type: %v\n", interactionType)
			fmt.Printf("     TX Details: %v\n", txDetails)
		}
		if count == 0 {
			fmt.Println("   No ERC20_TRANSFER relationships found")
		}
	} else {
		fmt.Printf("   Error: %v\n", err)
	}

	fmt.Println("\n✅ Verification completed!")
}
