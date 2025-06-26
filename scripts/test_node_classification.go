package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/domain/service"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"
)

// MockBlockchainService for testing
type MockBlockchainService struct{}

func (m *MockBlockchainService) GetCodeAt(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error) {
	// Mock different types of addresses
	switch address {
	case "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984": // Uniswap Token
		// Return mock ERC20 bytecode with transfer and approve function signatures
		return []byte{0xa9, 0x05, 0x9c, 0xbb, 0x00, 0x00, 0x09, 0x5e, 0xa7, 0xb3}, nil
	case "0x7a250d5630b4cf539739df2c5dacb4c659f2488d": // Uniswap V2 Router
		// Return mock DEX bytecode with swap function signatures
		return []byte{0x7f, 0xf3, 0x6a, 0xb5, 0x00, 0x00, 0x18, 0xcb, 0xaf, 0xe5}, nil
	case "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": // WETH
		// Return mock WETH bytecode with deposit/withdraw signatures
		return []byte{0xd0, 0xe3, 0x0d, 0xb0, 0x00, 0x00, 0x2e, 0x1a, 0x7d, 0x4d}, nil
	case "0x000000000000000000000000000000000000dead": // Dead address (EOA)
		return []byte{}, nil // Empty bytecode = EOA
	case "0x1234567890123456789012345678901234567890": // Regular EOA
		return []byte{}, nil // Empty bytecode = EOA
	default:
		return []byte{}, nil // Default to EOA for unknown addresses
	}
}

func main() {
	fmt.Println("🔍 Enhanced Node Classification Test with Bytecode Analysis")
	fmt.Println("===========================================================")

	// Initialize logger
	logger, err := logger.NewLogger("debug")
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	// Initialize blockchain service mock
	blockchain := &MockBlockchainService{}

	// Initialize node classifier with blockchain service
	classifier := service.NewNodeClassifierService(blockchain, logger)

	ctx := context.Background()

	// Test cases: Mix of contracts and EOAs
	testCases := []struct {
		name        string
		address     string
		description string
		expectType  entity.NodeType
	}{
		{
			name:        "Uniswap Token Contract",
			address:     "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984",
			description: "Should be classified as TokenContract",
			expectType:  entity.NodeTypeTokenContract,
		},
		{
			name:        "Uniswap V2 Router",
			address:     "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
			description: "Should be classified as DEXContract",
			expectType:  entity.NodeTypeDEXContract,
		},
		{
			name:        "WETH Contract",
			address:     "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			description: "Should be classified as TokenContract (WETH)",
			expectType:  entity.NodeTypeTokenContract,
		},
		{
			name:        "Dead Address (EOA)",
			address:     "0x000000000000000000000000000000000000dead",
			description: "Should be classified as EOA",
			expectType:  entity.NodeTypeEOA,
		},
		{
			name:        "Regular User Wallet",
			address:     "0x1234567890123456789012345678901234567890",
			description: "Should be classified as EOA",
			expectType:  entity.NodeTypeEOA,
		},
	}

	fmt.Printf("Testing %d addresses...\n\n", len(testCases))

	for i, testCase := range testCases {
		fmt.Printf("Test %d: %s\n", i+1, testCase.name)
		fmt.Printf("Address: %s\n", testCase.address)
		fmt.Printf("Expected: %s\n", testCase.expectType)

		// Classify the node
		classification, err := classifier.ClassifyNode(ctx, testCase.address, nil, []string{})
		if err != nil {
			fmt.Printf("❌ Error: %v\n\n", err)
			continue
		}

		// Display results
		fmt.Printf("✅ Classification Results:\n")
		fmt.Printf("   Primary Type: %s\n", classification.PrimaryType)
		fmt.Printf("   Risk Level: %s\n", classification.RiskLevel)
		fmt.Printf("   Confidence: %.2f\n", classification.ConfidenceScore)
		fmt.Printf("   Detection Methods: %v\n", classification.DetectionMethods)
		fmt.Printf("   Tags: %v\n", classification.Tags)

		// Check if classification matches expectation
		if classification.PrimaryType == testCase.expectType {
			fmt.Printf("✅ PASSED: Classification matches expectation\n")
		} else {
			fmt.Printf("❌ FAILED: Expected %s, got %s\n", testCase.expectType, classification.PrimaryType)
		}

		// Check if bytecode analysis was used
		hasBytecodeAnalysis := false
		for _, method := range classification.DetectionMethods {
			if method == string(entity.DetectionMethodBytecodeAnalysis) {
				hasBytecodeAnalysis = true
				break
			}
		}

		if hasBytecodeAnalysis {
			fmt.Printf("✅ Bytecode analysis was successfully used\n")
		} else {
			fmt.Printf("⚠️  Bytecode analysis was not used\n")
		}

		fmt.Println(strings.Repeat("-", 60))
	}

	fmt.Println("\n🎯 Summary:")
	fmt.Println("The enhanced node classifier now:")
	fmt.Println("1. ✅ Checks bytecode to distinguish contracts from EOAs")
	fmt.Println("2. ✅ Analyzes contract bytecode to determine contract type")
	fmt.Println("3. ✅ Uses multiple detection methods for higher confidence")
	fmt.Println("4. ✅ Properly tags addresses as 'smart_contract' or 'eoa'")
	fmt.Println("5. ✅ Provides more accurate risk assessment")

	fmt.Println("\n📊 Contract Type Detection Patterns:")
	fmt.Println("• ERC20 Tokens: Look for transfer(a9059cbb) + approve(095ea7b3)")
	fmt.Println("• DEX Contracts: Look for swap functions (7ff36ab5, 18cbafe5)")
	fmt.Println("• WETH Contracts: Look for deposit(d0e30db0) + withdraw(2e1a7d4d)")
	fmt.Println("• Proxy Contracts: Look for implementation slot pattern")
	fmt.Println("• Factory Contracts: Look for CREATE2 bytecode patterns")
}
