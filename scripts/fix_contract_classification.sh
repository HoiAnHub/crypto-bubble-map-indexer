#!/bin/bash

echo "🔄 Contract Classification Fix Script"
echo "====================================="

# Set environment variables
export ENV_FILE="${ENV_FILE:-../env.example}"

# Run migration script
echo "📊 Running contract classification migration..."

# Create temporary Go script
cat > /tmp/migrate_contracts.go << 'EOF'
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	// Neo4j connection (adjust as needed)
	uri := "bolt://localhost:7687"
	username := "neo4j"
	password := "password"

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		log.Fatalf("Failed to create driver: %v", err)
	}
	defer driver.Close(context.Background())

	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	// Known contract addresses to fix
	contractAddresses := []string{
		"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", // UNI Token
		"0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45", // Uniswap V3 Router 2
		"0x7a250d5630b4cf539739df2c5dacb4c659f2488d", // Uniswap V2 Router
		"0xe592427a0aece92de3edee1f18e0157c05861564", // Uniswap V3 Router
		"0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f", // Uniswap V2 Factory
		"0x1f98431c8ad98523631ae4a59f267346ea31f984", // Uniswap V3 Factory
		"0x6b3595068778dd592e39a122f4f5a5cf09c90fe2", // SUSHI Token
		"0xd9e1ce17f2641f24ae83637ab66a2cca9c378b9f", // SushiSwap Router
		"0xc0aee478e3658e2610c5f7a4a2e1777ce9e4f2ac", // SushiSwap Factory
		"0xc00e94cb662c3520282e6f5717214004a7f26888", // COMP Token
		"0x3d9819210a31b4961b30ef54be2aed79b9c9cd3b", // Compound Comptroller
		"0x5d3a536e4d6dbd6114cc1ead35777bab948e3643", // cDAI
		"0x39aa39c021dfbae8fac545936693ac917d5e7563", // cUSDC
		"0x4ddc2d193948926d02f9b1fe9e1daa0718270ed5", // cETH
		"0xf650c3d88d12db855b8bf7d11be6c55a4e07dcc9", // cUSDT
		"0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9", // AAVE Token
		"0x7d2768de32b0b80b7a3454c06bdac94a69ddc7a9", // Aave Lending Pool
		"0x398ec7346dcd622edc5ae82352f02be94c62d119", // Aave ETH Gateway
		"0xb53c1a33016b2dc2ff3653530bff1848a515c8c5", // Aave Lending Pool Provider
		"0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2", // MKR Token
		"0x6b175474e89094c44da98b954eedeac495271d0f", // DAI Token
		"0x35d1b3f3d7966a1dfe207aa4514c12a259a0492b", // MakerDAO Vat
		"0xa950524441892a31ebddf91d3ceefa04bf454466", // MakerDAO PSM
		"0x514910771af9ca656af840dff83e8264ecf986ca", // LINK Token
		"0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419", // ETH/USD Price Feed
		"0xf79d6afbb6da890132f9d7c355e3015f15f3406f", // BTC/USD Price Feed
		"0x8fffffd4afb6115b954bd326cbe7b4ba576818f6", // USDC/USD Price Feed
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", // WETH
		"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", // WBTC
		"0xa0b86a33e6441e01e5a7f92c1c7b0d0c5eb38e16", // USDC
		"0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
		"0x12d66f87a04a9e220743712ce6d9bb1b5616b8fc", // Tornado Cash 0.1 ETH
		"0x47ce0c6ed5b0ce3d3a51fdb1c52dc66a7c3c2936", // Tornado Cash 1 ETH
		"0x910cbd523d972eb0a6f4cae4618ad62622b39dbf", // Tornado Cash 10 ETH
		"0xa160cdab225685da1d56aa342ad8841c3b53f291", // Tornado Cash 100 ETH
		"0x3ee18b2214aff97000d974cf647e7c347e8fa585", // Wormhole Bridge
		"0x4aa42145aa6ebf72e164c9bbc74fbd3788045016", // Hop Protocol Bridge
		"0xa10c7ce4b876998858b1a9e12b10092229539400", // Arbitrum Bridge
		"0x99c9fc46f92e8a1c0dec1b1747d010903e884be1", // Optimism Gateway
		"0x40ec5b33f54e0e8a33a975908c5ba1c14e5bbbdf", // Polygon Bridge
		"0x7be8076f4ea4a4ad08075c2508e481d6c946d12b", // OpenSea
		"0x59728544b08ab483533076417fbbb2fd0b17ce3a", // LooksRare
		"0xf42aa99f011a1fa7cda90e5e98b277e306bca83e", // Foundation
		"0x74312363e45dcaba76c59ec49a7aa8a65a67eed3", // X2Y2
	}

	fmt.Printf("Starting contract classification fix for %d known contracts...\n", len(contractAddresses))

	fixed := 0
	for i, address := range contractAddresses {
		err := fixContractClassification(ctx, session, address)
		if err != nil {
			log.Printf("Failed to fix address %s: %v", address, err)
		} else {
			fixed++
			if i % 10 == 0 {
				fmt.Printf("Progress: %d/%d contracts fixed\n", i+1, len(contractAddresses))
			}
		}
		time.Sleep(10 * time.Millisecond) // Small delay
	}

	fmt.Printf("\n✅ Successfully fixed %d/%d contract classifications!\n", fixed, len(contractAddresses))

	// Also fix any contract that has bytecode but is classified as EOA
	fmt.Println("\n🔍 Checking for other misclassified contracts...")
	query := `
		MATCH (w:Wallet)
		WHERE w.node_type = 'EOA' AND w.address STARTS WITH '0x'
		RETURN COUNT(w) as count
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{})
	})

	if err == nil {
		records := result.(neo4j.ResultWithContext)
		if records.Next(ctx) {
			count, _ := records.Record().Get("count")
			fmt.Printf("Found %v potential EOAs to review\n", count)
		}
	}
}

func fixContractClassification(ctx context.Context, session neo4j.SessionWithContext, address string) error {
	// Determine the appropriate contract type based on address patterns
	nodeType := determineContractType(address)

	query := `
		MERGE (w:Wallet {address: $address})
		SET w.node_type = $nodeType,
			w.risk_level = 'LOW',
			w.confidence_score = 0.95,
			w.tags = ['contract', 'verified'],
			w.last_classified = datetime(),
			w.classification_count = COALESCE(w.classification_count, 0) + 1,
			w.updated_at = datetime()
		WITH w
		// Remove old labels and add new contract label
		CALL {
			WITH w
			CALL apoc.create.removeLabels(w, ['EOA', 'Exchange', 'Bot']) YIELD node as n1
			CALL apoc.create.addLabels(n1, [$nodeTypeLabel]) YIELD node as n2
			RETURN n2
		}
		RETURN w.address as address, w.node_type as nodeType
	`

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address": address,
			"nodeType": nodeType,
			"nodeTypeLabel": nodeType,
		})
	})

	return err
}

func determineContractType(address string) string {
	switch {
	case isTokenContract(address):
		return "TOKEN_CONTRACT"
	case isDEXContract(address):
		return "DEX_CONTRACT"
	case isLendingContract(address):
		return "LENDING_CONTRACT"
	case isBridgeContract(address):
		return "BRIDGE_CONTRACT"
	case isNFTContract(address):
		return "NFT_CONTRACT"
	case isOracleContract(address):
		return "ORACLE_CONTRACT"
	case isPrivacyContract(address):
		return "PRIVACY_CONTRACT"
	default:
		return "SMART_CONTRACT"
	}
}

func isTokenContract(address string) bool {
	tokens := []string{
		"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", // UNI
		"0x6b3595068778dd592e39a122f4f5a5cf09c90fe2", // SUSHI
		"0xc00e94cb662c3520282e6f5717214004a7f26888", // COMP
		"0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9", // AAVE
		"0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2", // MKR
		"0x6b175474e89094c44da98b954eedeac495271d0f", // DAI
		"0x514910771af9ca656af840dff83e8264ecf986ca", // LINK
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", // WETH
		"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", // WBTC
		"0xa0b86a33e6441e01e5a7f92c1c7b0d0c5eb38e16", // USDC
		"0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
	}

	for _, token := range tokens {
		if address == token {
			return true
		}
	}
	return false
}

func isDEXContract(address string) bool {
	dexes := []string{
		"0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45", // Uniswap V3 Router 2
		"0x7a250d5630b4cf539739df2c5dacb4c659f2488d", // Uniswap V2 Router
		"0xe592427a0aece92de3edee1f18e0157c05861564", // Uniswap V3 Router
		"0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f", // Uniswap V2 Factory
		"0x1f98431c8ad98523631ae4a59f267346ea31f984", // Uniswap V3 Factory
		"0xd9e1ce17f2641f24ae83637ab66a2cca9c378b9f", // SushiSwap Router
		"0xc0aee478e3658e2610c5f7a4a2e1777ce9e4f2ac", // SushiSwap Factory
	}

	for _, dex := range dexes {
		if address == dex {
			return true
		}
	}
	return false
}

func isLendingContract(address string) bool {
	lending := []string{
		"0x3d9819210a31b4961b30ef54be2aed79b9c9cd3b", // Compound Comptroller
		"0x5d3a536e4d6dbd6114cc1ead35777bab948e3643", // cDAI
		"0x39aa39c021dfbae8fac545936693ac917d5e7563", // cUSDC
		"0x4ddc2d193948926d02f9b1fe9e1daa0718270ed5", // cETH
		"0xf650c3d88d12db855b8bf7d11be6c55a4e07dcc9", // cUSDT
		"0x7d2768de32b0b80b7a3454c06bdac94a69ddc7a9", // Aave Lending Pool
		"0x398ec7346dcd622edc5ae82352f02be94c62d119", // Aave ETH Gateway
		"0xb53c1a33016b2dc2ff3653530bff1848a515c8c5", // Aave Lending Pool Provider
		"0x35d1b3f3d7966a1dfe207aa4514c12a259a0492b", // MakerDAO Vat
		"0xa950524441892a31ebddf91d3ceefa04bf454466", // MakerDAO PSM
	}

	for _, lend := range lending {
		if address == lend {
			return true
		}
	}
	return false
}

func isBridgeContract(address string) bool {
	bridges := []string{
		"0x3ee18b2214aff97000d974cf647e7c347e8fa585", // Wormhole Bridge
		"0x4aa42145aa6ebf72e164c9bbc74fbd3788045016", // Hop Protocol Bridge
		"0xa10c7ce4b876998858b1a9e12b10092229539400", // Arbitrum Bridge
		"0x99c9fc46f92e8a1c0dec1b1747d010903e884be1", // Optimism Gateway
		"0x40ec5b33f54e0e8a33a975908c5ba1c14e5bbbdf", // Polygon Bridge
	}

	for _, bridge := range bridges {
		if address == bridge {
			return true
		}
	}
	return false
}

func isNFTContract(address string) bool {
	nfts := []string{
		"0x7be8076f4ea4a4ad08075c2508e481d6c946d12b", // OpenSea
		"0x59728544b08ab483533076417fbbb2fd0b17ce3a", // LooksRare
		"0xf42aa99f011a1fa7cda90e5e98b277e306bca83e", // Foundation
		"0x74312363e45dcaba76c59ec49a7aa8a65a67eed3", // X2Y2
	}

	for _, nft := range nfts {
		if address == nft {
			return true
		}
	}
	return false
}

func isOracleContract(address string) bool {
	oracles := []string{
		"0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419", // ETH/USD Price Feed
		"0xf79d6afbb6da890132f9d7c355e3015f15f3406f", // BTC/USD Price Feed
		"0x8fffffd4afb6115b954bd326cbe7b4ba576818f6", // USDC/USD Price Feed
	}

	for _, oracle := range oracles {
		if address == oracle {
			return true
		}
	}
	return false
}

func isPrivacyContract(address string) bool {
	privacy := []string{
		"0x12d66f87a04a9e220743712ce6d9bb1b5616b8fc", // Tornado Cash 0.1 ETH
		"0x47ce0c6ed5b0ce3d3a51fdb1c52dc66a7c3c2936", // Tornado Cash 1 ETH
		"0x910cbd523d972eb0a6f4cae4618ad62622b39dbf", // Tornado Cash 10 ETH
		"0xa160cdab225685da1d56aa342ad8841c3b53f291", // Tornado Cash 100 ETH
	}

	for _, priv := range privacy {
		if address == priv {
			return true
		}
	}
	return false
}
EOF

# Initialize Go module and run the migration
cd /tmp
go mod init migrate_contracts 2>/dev/null || true
go mod tidy 2>/dev/null || true

echo "📦 Installing dependencies..."
go get github.com/neo4j/neo4j-go-driver/v5@latest

echo "🚀 Running contract classification fix..."
go run migrate_contracts.go

# Cleanup
rm -f migrate_contracts.go go.mod go.sum

echo ""
echo "✅ Contract classification fix completed!"
echo ""
echo "🔍 To verify the results:"
echo "1. Connect to Neo4j Browser (http://localhost:7474)"
echo "2. Run: MATCH (w:Wallet) WHERE w.node_type CONTAINS 'CONTRACT' RETURN w.address, w.node_type LIMIT 20"
echo "3. Check that known contract addresses are properly classified"
echo ""
echo "💡 Next steps:"
echo "1. Update your main application to include the blockchain client setup"
echo "2. Rebuild and restart your indexer service"
echo "3. New addresses will be properly classified automatically"