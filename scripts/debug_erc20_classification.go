package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/infrastructure/blockchain"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"
)

func main() {
	fmt.Println("🔍 ERC20 Transfer Classification Debug")
	fmt.Println("=====================================")

	// Initialize logger
	logger, err := logger.NewLogger("debug")
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	// Initialize decoder
	decoder := blockchain.NewERC20DecoderService(logger)
	ctx := context.Background()

	// Test cases that should create ERC20_TRANSFER relationships
	testTransactions := []struct {
		name     string
		tx       *entity.Transaction
		expected entity.ContractInteractionType
	}{
		{
			name: "ERC20 Transfer",
			tx: &entity.Transaction{
				Hash:        "0x123456789",
				From:        "0x1111111111111111111111111111111111111111",
				To:          "0xa0b86a33e6441e01e5a7f92c1c7b0d0c5eb38e16",
				Value:       "1000000000000000000",
				Data:        "0xa9059cbb0000000000000000000000002222222222222222222222222222222222222222000000000000000000000000000000000000000000000000016345785d8a0000",
				BlockNumber: "12345",
				Network:     "ethereum",
			},
			expected: entity.InteractionTransfer,
		},
		{
			name: "ERC20 Approve",
			tx: &entity.Transaction{
				Hash:        "0x987654321",
				From:        "0x1111111111111111111111111111111111111111",
				To:          "0xa0b86a33e6441e01e5a7f92c1c7b0d0c5eb38e16",
				Value:       "0",
				Data:        "0x095ea7b30000000000000000000000003333333333333333333333333333333333333333ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
				BlockNumber: "12346",
				Network:     "ethereum",
			},
			expected: entity.InteractionApprove,
		},
		{
			name: "DEX Swap",
			tx: &entity.Transaction{
				Hash:        "0xabcdef123",
				From:        "0x1111111111111111111111111111111111111111",
				To:          "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
				Value:       "100000000000000000",
				Data:        "0x7ff36ab5000000000000000000000000000000000000000000000000000000000000008000000000000000000000000011111111111111111111111111111111111111110000000000000000000000000000000000000000000000000000000061a80d80",
				BlockNumber: "12347",
				Network:     "ethereum",
			},
			expected: entity.InteractionSwap,
		},
		{
			name: "WETH Deposit",
			tx: &entity.Transaction{
				Hash:        "0x456789abc",
				From:        "0x1111111111111111111111111111111111111111",
				To:          "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Value:       "1000000000000000000",
				Data:        "0xd0e30db0",
				BlockNumber: "12348",
				Network:     "ethereum",
			},
			expected: entity.InteractionDeposit,
		},
		{
			name: "Unknown Contract Call",
			tx: &entity.Transaction{
				Hash:        "0x999888777",
				From:        "0x1111111111111111111111111111111111111111",
				To:          "0x5555555555555555555555555555555555555555",
				Value:       "0",
				Data:        "0x12345678000000000000000000000000000000000000000000000000000000000000001234",
				BlockNumber: "12349",
				Network:     "ethereum",
			},
			expected: entity.InteractionUnknownContract,
		},
		{
			name: "ETH Transfer (no data)",
			tx: &entity.Transaction{
				Hash:        "0x111222333",
				From:        "0x1111111111111111111111111111111111111111",
				To:          "0x2222222222222222222222222222222222222222",
				Value:       "1000000000000000000",
				Data:        "",
				BlockNumber: "12350",
				Network:     "ethereum",
			},
			expected: entity.InteractionETHTransfer,
		},
	}

	fmt.Printf("Testing %d transactions...\n\n", len(testTransactions))

	for i, test := range testTransactions {
		fmt.Printf("=== Test %d: %s ===\n", i+1, test.name)
		fmt.Printf("Transaction Hash: %s\n", test.tx.Hash)
		fmt.Printf("From: %s\n", test.tx.From)
		fmt.Printf("To: %s\n", test.tx.To)
		fmt.Printf("Value: %s\n", test.tx.Value)
		fmt.Printf("Data: %s\n", test.tx.Data)
		fmt.Printf("Expected Type: %s\n", test.expected)

		// Decode the transaction
		transfers, err := decoder.DecodeERC20Transfer(ctx, test.tx)
		if err != nil {
			fmt.Printf("❌ Error decoding: %v\n\n", err)
			continue
		}

		if len(transfers) == 0 {
			fmt.Printf("⚠️  No transfers decoded\n\n")
			continue
		}

		// Check each transfer
		for j, transfer := range transfers {
			fmt.Printf("\n--- Transfer %d ---\n", j+1)
			fmt.Printf("Contract Address: %s\n", transfer.ContractAddress)
			fmt.Printf("From: %s\n", transfer.From)
			fmt.Printf("To: %s\n", transfer.To)
			fmt.Printf("Value: %s\n", transfer.Value)
			fmt.Printf("Interaction Type: %s\n", transfer.InteractionType)
			fmt.Printf("Method Signature: %s\n", transfer.MethodSignature)
			fmt.Printf("Success: %t\n", transfer.Success)

			// Check relationship type mapping
			relationshipType := transfer.InteractionType.GetRelationshipType()
			fmt.Printf("Relationship Type: %s\n", relationshipType)

			// Verify expected vs actual
			if transfer.InteractionType == test.expected {
				fmt.Printf("✅ PASSED: Interaction type matches expected\n")
			} else {
				fmt.Printf("❌ FAILED: Expected %s, got %s\n", test.expected, transfer.InteractionType)
			}

			// Check if it would go to CONTRACT_INTERACTION
			if relationshipType == "CONTRACT_INTERACTION" {
				fmt.Printf("⚠️  WARNING: This will be stored as CONTRACT_INTERACTION in Neo4j\n")
				fmt.Printf("   Reason: GetRelationshipType() returned default\n")
			} else {
				fmt.Printf("✅ Good: Will be stored as %s in Neo4j\n", relationshipType)
			}
		}

		fmt.Println(strings.Repeat("-", 60))
	}

	// Test the GetRelationshipType mapping directly
	fmt.Println("\n🔧 Direct Testing of GetRelationshipType():")
	fmt.Println("=========================================")

	allTypes := []entity.ContractInteractionType{
		entity.InteractionTransfer,
		entity.InteractionTransferFrom,
		entity.InteractionApprove,
		entity.InteractionIncreaseAllowance,
		entity.InteractionDecreaseAllowance,
		entity.InteractionSwap,
		entity.InteractionAddLiquidity,
		entity.InteractionRemoveLiquidity,
		entity.InteractionDeposit,
		entity.InteractionWithdraw,
		entity.InteractionMulticall,
		entity.InteractionETHTransfer,
		entity.InteractionUnknownContract,
	}

	for _, interactionType := range allTypes {
		relationshipType := interactionType.GetRelationshipType()
		fmt.Printf("%-25s → %s\n", string(interactionType), relationshipType)
	}

	fmt.Println("\n📊 Summary:")
	fmt.Println("If you're seeing too many CONTRACT_INTERACTION relationships, check:")
	fmt.Println("1. ✅ InteractionType is being set correctly in createXXXRecord functions")
	fmt.Println("2. ✅ GetRelationshipType() mapping covers all interaction types")
	fmt.Println("3. ✅ Transaction data is being parsed correctly")
	fmt.Println("4. ⚠️  Most likely: Many transactions have empty/missing InteractionType")
	fmt.Println("5. ⚠️  Solution: Ensure all create functions set InteractionType properly")
}
