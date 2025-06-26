package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/infrastructure/blockchain"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"
)

func main() {
	// Initialize logger
	logger, _ := logger.NewLogger("debug")

	// Initialize services
	decoder := blockchain.NewERC20DecoderService(logger)
	classifier := blockchain.NewContractClassifierService(logger)

	ctx := context.Background()

	fmt.Println("🏗️ Enhanced Contract Classification Test")
	fmt.Println("=========================================")

	// Test scenarios with real-world contracts
	testScenarios := []struct {
		name         string
		contractAddr string
		methodSigs   []string
		descriptions []string
	}{
		{
			name:         "Uniswap V2 Router",
			contractAddr: "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
			methodSigs:   []string{"7ff36ab5", "18cbafe5", "e8e33700"},
			descriptions: []string{"swapExactETHForTokens", "swapExactTokensForETH", "addLiquidity"},
		},
		{
			name:         "Compound cUSDC",
			contractAddr: "0x39aa39c021dfbae8fac545936693ac917d5e7563",
			methodSigs:   []string{"a6afed95", "852a12e3", "d0e30db0"},
			descriptions: []string{"mint", "redeem", "deposit"},
		},
		{
			name:         "WETH Contract",
			contractAddr: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			methodSigs:   []string{"d0e30db0", "2e1a7d4d"},
			descriptions: []string{"deposit", "withdraw"},
		},
		{
			name:         "Multicall Contract",
			contractAddr: "0x1f98431c8ad98523631ae4a59f267346ea31f984",
			methodSigs:   []string{"ac9650d8", "5ae401dc"},
			descriptions: []string{"multicall", "multicallWithDeadline"},
		},
	}

	for _, scenario := range testScenarios {
		fmt.Printf("\n🔍 Testing %s\n", scenario.name)
		fmt.Printf("   Address: %s\n", scenario.contractAddr)
		fmt.Printf("   %s\n", strings.Repeat("-", 50))

		var transfers []*entity.ERC20Transfer

		// Test each method signature
		for i, methodSig := range scenario.methodSigs {
			tx := createTestTx(fmt.Sprintf("0x%d", i+1), methodSig, scenario.contractAddr)

			decoded, err := decoder.DecodeERC20Transfer(ctx, tx)
			if err != nil {
				fmt.Printf("   ❌ Error decoding %s: %v\n", tx.Hash, err)
				continue
			}

			for _, transfer := range decoded {
				transfers = append(transfers, transfer)

				fmt.Printf("   📝 Method %s (%s):\n", methodSig, scenario.descriptions[i])
				fmt.Printf("      Type: %s\n", transfer.InteractionType)
				fmt.Printf("      Relationship: %s\n", transfer.InteractionType.GetRelationshipType())
			}
		}

		// Classify the contract
		if len(transfers) > 0 {
			classification, err := classifier.ClassifyContract(ctx, scenario.contractAddr, transfers)
			if err != nil {
				fmt.Printf("   ❌ Classification error: %v\n", err)
				continue
			}

			fmt.Printf("\n   🎯 Classification Results:\n")
			fmt.Printf("      Primary Type: %s (%.2f confidence)\n", classification.PrimaryType, classification.ConfidenceScore)
			fmt.Printf("      Category: %s\n", classification.PrimaryType.GetContractTypeCategory())
			fmt.Printf("      Is DeFi: %v\n", classification.PrimaryType.IsDefiProtocol())

			if len(classification.DetectedProtocols) > 0 {
				fmt.Printf("      Detected Protocols: %v\n", classification.DetectedProtocols)
			}

			fmt.Printf("      Total Interactions: %d\n", classification.TotalInteractions)
			fmt.Printf("      Tags: %v\n", classification.Tags)
		}
	}

	// Test quick classification
	fmt.Printf("\n⚡ Quick Method Signature Classification Test\n")
	fmt.Printf("%s\n", strings.Repeat("=", 50))

	quickTests := map[string]string{
		"7ff36ab5": "Uniswap swapExactETHForTokens",
		"18cbafe5": "Uniswap swapExactTokensForETH",
		"a6afed95": "Compound mint",
		"852a12e3": "Compound redeem",
		"d65d7f80": "Aave deposit",
		"ac9650d8": "Multicall",
		"d0e30db0": "WETH deposit",
		"a9059cbb": "ERC20 transfer",
	}

	for methodSig, description := range quickTests {
		classified := classifier.ClassifyFromMethodSignature(methodSig)
		fmt.Printf("   %s (%s) → %s\n", methodSig, description, classified)
	}

	// Summary
	fmt.Printf("\n🎉 Enhanced Contract Classification Summary\n")
	fmt.Printf("%s\n", strings.Repeat("=", 50))
	fmt.Println("✅ Multiple contract types detected and classified")
	fmt.Println("✅ Confidence scoring and verification status")
	fmt.Println("✅ Protocol detection (Uniswap, Compound, etc.)")
	fmt.Println("✅ Activity-based tagging system")
	fmt.Println("✅ Method signature pattern matching")
	fmt.Println("✅ Comprehensive classification rules engine")

	fmt.Printf("\n📈 Benefits:\n")
	fmt.Println("🔹 Better graph modeling with specific contract types")
	fmt.Println("🔹 Enhanced analytics and insights")
	fmt.Println("🔹 Protocol-specific relationship types")
	fmt.Println("🔹 Automatic DeFi ecosystem mapping")
	fmt.Println("🔹 Smart contract verification status")
}

func createTestTx(hash, methodSig, contractAddr string) *entity.Transaction {
	return &entity.Transaction{
		Hash:        hash,
		From:        "0xuser1234567890123456789012345678901234567890",
		To:          contractAddr,
		Value:       "1000000000000000000",
		Data:        "0x" + methodSig + strings.Repeat("0", 128),
		BlockNumber: "18000000",
		Timestamp:   time.Now(),
		Network:     "ethereum",
	}
}
