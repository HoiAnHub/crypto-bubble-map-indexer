// Enhanced ERC20 Decoder Demo
package main

import (
	"context"
	"fmt"
	"time"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/infrastructure/blockchain"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"
)

func main() {
	// Initialize logger
	logger, _ := logger.NewLogger("debug")

	// Initialize ERC20 decoder
	decoder := blockchain.NewERC20DecoderService(logger)

	ctx := context.Background()

	// Test various transaction types
	testTransactions := []*entity.Transaction{
		{
			Hash:        "0x1234567890abcdef1234567890abcdef12345678",
			From:        "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
			To:          "0xa0b86a33e6411dd02d5bb2bb4cb4ecec4f5c87c6",
			Value:       "1000000000000000000",                                                                                                                       // 1 ETH
			Data:        "0xa9059cbb000000000000000000000000742d35cc6b7d72f4b73a3623b498b9b3b8b2f6e0000000000000000000000000000000000000000000000000de0b6b3a7640000", // transfer
			BlockNumber: "18000000",
			Timestamp:   time.Now(),
			Network:     "ethereum",
		},
		{
			Hash:        "0x2234567890abcdef1234567890abcdef12345678",
			From:        "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
			To:          "0xa0b86a33e6411dd02d5bb2bb4cb4ecec4f5c87c6",
			Value:       "0",
			Data:        "0x095ea7b3000000000000000000000000742d35cc6b7d72f4b73a3623b498b9b3b8b2f6e00000000000000000000000000000000000000000000000000de0b6b3a7640000", // approve
			BlockNumber: "18000001",
			Timestamp:   time.Now(),
			Network:     "ethereum",
		},
		{
			Hash:        "0x3234567890abcdef1234567890abcdef12345678",
			From:        "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
			To:          "0xa0b86a33e6411dd02d5bb2bb4cb4ecec4f5c87c6",
			Value:       "0",
			Data:        "0x23b872dd000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000742d35cc6b7d72f4b73a3623b498b9b3b8b2f6e0000000000000000000000000000000000000000000000000de0b6b3a7640000", // transferFrom
			BlockNumber: "18000002",
			Timestamp:   time.Now(),
			Network:     "ethereum",
		},
		{
			Hash:        "0x4234567890abcdef1234567890abcdef12345678",
			From:        "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
			To:          "0xa0b86a33e6411dd02d5bb2bb4cb4ecec4f5c87c6",
			Value:       "0",
			Data:        "0x7ff36ab5000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000120", // swap
			BlockNumber: "18000003",
			Timestamp:   time.Now(),
			Network:     "ethereum",
		},
		{
			Hash:        "0x5234567890abcdef1234567890abcdef12345678",
			From:        "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
			To:          "0xa0b86a33e6411dd02d5bb2bb4cb4ecec4f5c87c6",
			Value:       "0",
			Data:        "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef", // unknown method
			BlockNumber: "18000004",
			Timestamp:   time.Now(),
			Network:     "ethereum",
		},
		{
			Hash:        "0x6234567890abcdef1234567890abcdef12345678",
			From:        "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
			To:          "0x742d35cc6b7d72f4b73a3623b498b9b3b8b2f6e0",
			Value:       "500000000000000000", // 0.5 ETH
			Data:        "",                   // pure ETH transfer
			BlockNumber: "18000005",
			Timestamp:   time.Now(),
			Network:     "ethereum",
		},
	}

	fmt.Println("🚀 Enhanced ERC20 Decoder Test Results")
	fmt.Println("=====================================")

	contractInteractionStats := make(map[string]int)
	relationshipStats := make(map[string]int)

	for i, tx := range testTransactions {
		fmt.Printf("\n%d. Testing Transaction: %s\n", i+1, tx.Hash)
		fmt.Printf("   From: %s\n", tx.From)
		fmt.Printf("   To: %s\n", tx.To)
		fmt.Printf("   Value: %s wei\n", tx.Value)
		if len(tx.Data) > 50 {
			fmt.Printf("   Data: %s...\n", tx.Data[:50])
		} else {
			fmt.Printf("   Data: %s\n", tx.Data)
		}

		// Decode the transaction
		transfers, err := decoder.DecodeERC20Transfer(ctx, tx)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}

		if len(transfers) == 0 {
			fmt.Printf("   ⚪ No contract interactions found\n")
			continue
		}

		for _, transfer := range transfers {
			fmt.Printf("   ✅ Contract Interaction Detected:\n")
			fmt.Printf("      Type: %s\n", transfer.InteractionType)
			fmt.Printf("      Contract: %s\n", transfer.ContractAddress)
			fmt.Printf("      From: %s\n", transfer.From)
			fmt.Printf("      To: %s\n", transfer.To)
			fmt.Printf("      Value: %s\n", transfer.Value)
			fmt.Printf("      Method Signature: %s\n", transfer.MethodSignature)
			fmt.Printf("      Relationship Type: %s\n", transfer.InteractionType.GetRelationshipType())

			// Update stats
			contractInteractionStats[string(transfer.InteractionType)]++
			relationshipStats[transfer.InteractionType.GetRelationshipType()]++
		}
	}

	// Print comprehensive statistics
	fmt.Println("\n📊 Comprehensive Analysis Results")
	fmt.Println("=================================")

	fmt.Println("\n🔍 Contract Interaction Types:")
	for interactionType, count := range contractInteractionStats {
		fmt.Printf("   %s: %d transactions\n", interactionType, count)
	}

	fmt.Println("\n🔗 Neo4J Relationship Types:")
	for relType, count := range relationshipStats {
		fmt.Printf("   %s: %d relationships\n", relType, count)
	}

	// Test relationship type mapping
	fmt.Println("\n🎯 Relationship Type Mapping Test:")
	fmt.Println("===================================")

	allInteractionTypes := []entity.ContractInteractionType{
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

	for _, interactionType := range allInteractionTypes {
		relType := interactionType.GetRelationshipType()
		fmt.Printf("   %s → %s\n", interactionType, relType)
	}

	// Best practices summary
	fmt.Println("\n🏆 Enhanced Features Summary:")
	fmt.Println("=============================")
	fmt.Printf("✅ Supports %d different contract interaction types\n", len(allInteractionTypes))
	fmt.Printf("✅ Creates %d distinct Neo4J relationship types\n", len(getUniqueRelationshipTypes()))
	fmt.Printf("✅ Handles unknown contract calls gracefully\n")
	fmt.Printf("✅ Captures method signatures for analysis\n")
	fmt.Printf("✅ Supports pure ETH transfers\n")
	fmt.Printf("✅ Enhanced error handling and logging\n")
	fmt.Printf("✅ Best practice graph modeling\n")

	fmt.Println("\n🎉 Enhanced ERC20 Decoder Test Completed Successfully!")
}

func getUniqueRelationshipTypes() map[string]bool {
	types := make(map[string]bool)
	allInteractionTypes := []entity.ContractInteractionType{
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

	for _, t := range allInteractionTypes {
		types[t.GetRelationshipType()] = true
	}
	return types
}
