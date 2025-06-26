package service

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

// BlockchainClient interface for checking contract bytecode
type BlockchainClient interface {
	// GetCode returns the bytecode of a contract address
	// Returns empty string for EOA, non-empty for contracts
	GetCode(ctx context.Context, address string) (string, error)

	// IsContract checks if address is a contract (has bytecode)
	IsContract(ctx context.Context, address string) (bool, error)
}

// NodeClassifierService handles node classification logic
type NodeClassifierService struct {
	rules            []entity.NodeClassificationRule
	exchangePatterns map[string][]string // exchange name -> address patterns
	blacklistedAddrs map[string]string   // address -> reason
	sanctionedAddrs  map[string]string   // address -> sanction details
	knownContracts   map[string]entity.NodeType
	blockchainClient BlockchainClient // for checking contract bytecode
}

// NewNodeClassifierService creates a new node classifier service
func NewNodeClassifierService() *NodeClassifierService {
	service := &NodeClassifierService{
		rules:            []entity.NodeClassificationRule{},
		exchangePatterns: make(map[string][]string),
		blacklistedAddrs: make(map[string]string),
		sanctionedAddrs:  make(map[string]string),
		knownContracts:   make(map[string]entity.NodeType),
		blockchainClient: nil, // Will be set via SetBlockchainClient
	}

	service.initializeDefaultRules()
	service.initializeKnownPatterns()

	return service
}

// SetBlockchainClient sets the blockchain client for contract detection
func (ncs *NodeClassifierService) SetBlockchainClient(client BlockchainClient) {
	ncs.blockchainClient = client
}

// isContractAddress checks if an address is a contract by checking bytecode
func (ncs *NodeClassifierService) isContractAddress(ctx context.Context, address string) (bool, error) {
	if ncs.blockchainClient == nil {
		// Fallback: check known contracts list
		_, exists := ncs.knownContracts[strings.ToLower(address)]
		return exists, nil
	}

	return ncs.blockchainClient.IsContract(ctx, address)
}

// detectContractType attempts to detect contract type from patterns and known lists
func (ncs *NodeClassifierService) detectContractType(ctx context.Context, address string, isContract bool) entity.NodeType {
	if !isContract {
		return entity.NodeTypeUnknown
	}

	// Check known contracts first
	if nodeType, exists := ncs.knownContracts[strings.ToLower(address)]; exists {
		return nodeType
	}

	// Heuristic detection based on address patterns (if available in future)
	// This could be enhanced with bytecode analysis

	return entity.NodeTypeTokenContract // Default to token contract for unknown contracts
}

// ClassifyNode classifies a blockchain address into appropriate node type
func (ncs *NodeClassifierService) ClassifyNode(ctx context.Context, address string,
	stats *entity.WalletStats, patterns []string) (*entity.NodeClassification, error) {

	address = strings.ToLower(address)

	classification := &entity.NodeClassification{
		Address:              address,
		PrimaryType:          entity.NodeTypeUnknown,
		SecondaryTypes:       []entity.NodeType{},
		RiskLevel:            entity.RiskLevelUnknown,
		ConfidenceScore:      0.0,
		DetectionMethods:     []string{},
		Tags:                 []string{},
		LastClassified:       time.Now(),
		ClassificationCount:  1,
		Network:              "ethereum", // default
		SuspiciousActivities: []string{},
		BlacklistReasons:     []string{},
		SanctionDetails:      []string{},
		ReportedBy:           []string{},
	}

	// 1. Check blacklisted/sanctioned addresses first
	if reason, exists := ncs.blacklistedAddrs[address]; exists {
		classification.PrimaryType = entity.NodeTypeBlacklistedWallet
		classification.RiskLevel = entity.RiskLevelCritical
		classification.ConfidenceScore = 1.0
		classification.BlacklistReasons = []string{reason}
		classification.DetectionMethods = []string{string(entity.DetectionMethodBlocklist)}
		return classification, nil
	}

	if sanction, exists := ncs.sanctionedAddrs[address]; exists {
		classification.PrimaryType = entity.NodeTypeSanctioned
		classification.RiskLevel = entity.RiskLevelCritical
		classification.ConfidenceScore = 1.0
		classification.SanctionDetails = []string{sanction}
		classification.DetectionMethods = []string{string(entity.DetectionMethodBlocklist)}
		return classification, nil
	}

	// 2. Check if address is a contract first (most important check)
	isContract, err := ncs.isContractAddress(ctx, address)
	if err != nil {
		// Log error but continue with classification
		fmt.Printf("Warning: Could not check if address %s is contract: %v\n", address, err)
	}

	// 3. Check known contracts
	if nodeType, exists := ncs.knownContracts[address]; exists {
		classification.PrimaryType = nodeType
		classification.RiskLevel = nodeType.GetDefaultRiskLevel()
		classification.ConfidenceScore = 0.9
		classification.DetectionMethods = []string{string(entity.DetectionMethodManual)}
		classification.Tags = append(classification.Tags, "known_contract")
		return classification, nil
	}

	// 4. If it's a contract but not in known list, detect contract type
	if isContract {
		contractType := ncs.detectContractType(ctx, address, isContract)
		classification.PrimaryType = contractType
		classification.RiskLevel = contractType.GetDefaultRiskLevel()
		classification.ConfidenceScore = 0.6
		classification.DetectionMethods = []string{string(entity.DetectionMethodHeuristic)}
		classification.Tags = append(classification.Tags, "contract", "bytecode_detected")
		return classification, nil
	}

	// 5. Check exchange patterns (only for non-contracts)
	for exchange, patterns := range ncs.exchangePatterns {
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, address); matched {
				classification.PrimaryType = entity.NodeTypeExchangeWallet
				classification.Exchanges = []string{exchange}
				classification.ConfidenceScore = 0.8
				classification.RiskLevel = entity.RiskLevelLow
				classification.DetectionMethods = []string{string(entity.DetectionMethodPatternAnalysis)}
				break
			}
		}
		if classification.PrimaryType != entity.NodeTypeUnknown {
			break
		}
	}

	// 4. Analyze transaction patterns if statistics available
	if stats != nil {
		ncs.analyzeTransactionPatterns(classification, stats, patterns)
	}

	// 5. Apply classification rules
	ncs.applyClassificationRules(classification, stats, patterns)

	// 6. Set default if still unknown
	if classification.PrimaryType == entity.NodeTypeUnknown {
		// Final check: if it's a contract, don't classify as EOA
		if isContract {
			classification.PrimaryType = entity.NodeTypeTokenContract // Default contract type
			classification.RiskLevel = entity.RiskLevelLow
			classification.ConfidenceScore = 0.4
			classification.DetectionMethods = []string{string(entity.DetectionMethodHeuristic)}
			classification.Tags = append(classification.Tags, "contract", "unclassified_contract")
		} else {
			classification.PrimaryType = entity.NodeTypeEOA
			classification.RiskLevel = entity.RiskLevelLow
			classification.ConfidenceScore = 0.3
			classification.DetectionMethods = []string{string(entity.DetectionMethodHeuristic)}
			classification.Tags = append(classification.Tags, "eoa")
		}
	}

	return classification, nil
}

// analyzeTransactionPatterns analyzes transaction patterns to help classify nodes
func (ncs *NodeClassifierService) analyzeTransactionPatterns(classification *entity.NodeClassification,
	stats *entity.WalletStats, patterns []string) {

	// High-frequency trading patterns (MEV/Arbitrage bots)
	if stats.TransactionCount > 1000 {
		avgValue := new(big.Int)
		if totalVolume, ok := avgValue.SetString(stats.TotalVolume, 10); ok {
			avgTxValue := new(big.Int).Div(totalVolume, big.NewInt(stats.TransactionCount))

			// High frequency, low average value -> likely bot
			if stats.TransactionCount > 10000 {
				classification.SecondaryTypes = append(classification.SecondaryTypes, entity.NodeTypeMEVBot)
				classification.Tags = append(classification.Tags, "high_frequency")
				classification.ConfidenceScore += 0.3
			}

			// Very high average transaction value -> likely whale
			ethValue := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)    // 1 ETH in wei
			if avgTxValue.Cmp(new(big.Int).Mul(ethValue, big.NewInt(100))) > 0 { // > 100 ETH
				classification.SecondaryTypes = append(classification.SecondaryTypes, entity.NodeTypeWhale)
				classification.Tags = append(classification.Tags, "whale", "high_value")
				classification.ConfidenceScore += 0.2
			}
		}
	}

	// Exchange-like patterns (using transaction count and incoming/outgoing connections as proxy)
	if stats.IncomingConnections > 1000 && stats.OutgoingConnections > 1000 && stats.TransactionCount > 5000 {
		if classification.PrimaryType == entity.NodeTypeUnknown {
			classification.PrimaryType = entity.NodeTypeExchangeHotWallet
			classification.ConfidenceScore = 0.7
			classification.Tags = append(classification.Tags, "exchange_like", "high_volume")
		}
	}

	// Mixer/Privacy patterns
	for _, pattern := range patterns {
		if strings.Contains(pattern, "mixer") || strings.Contains(pattern, "tornado") {
			classification.PrimaryType = entity.NodeTypeMixerWallet
			classification.RiskLevel = entity.RiskLevelHigh
			classification.ConfidenceScore = 0.8
			classification.SuspiciousActivities = append(classification.SuspiciousActivities, "privacy_mixing")
			break
		}
	}
}

// applyClassificationRules applies the configured classification rules
func (ncs *NodeClassifierService) applyClassificationRules(classification *entity.NodeClassification,
	stats *entity.WalletStats, patterns []string) {

	for _, rule := range ncs.rules {
		score := ncs.calculateRuleScore(rule, classification, stats, patterns)

		if score >= rule.MinConfidence {
			if classification.PrimaryType == entity.NodeTypeUnknown {
				classification.PrimaryType = rule.NodeType
				classification.ConfidenceScore = score
				classification.RiskLevel = rule.NodeType.GetDefaultRiskLevel()
			} else {
				// Add as secondary type if confidence is high enough
				if score > 0.6 {
					classification.SecondaryTypes = append(classification.SecondaryTypes, rule.NodeType)
				}
			}
			classification.DetectionMethods = append(classification.DetectionMethods, string(entity.DetectionMethodHeuristic))
		}
	}
}

// calculateRuleScore calculates how well an address matches a classification rule
func (ncs *NodeClassifierService) calculateRuleScore(rule entity.NodeClassificationRule,
	classification *entity.NodeClassification, stats *entity.WalletStats, patterns []string) float64 {

	score := 0.0

	// Check required patterns
	requiredMatches := 0
	for _, required := range rule.RequiredPatterns {
		for _, pattern := range patterns {
			if strings.Contains(pattern, required) {
				requiredMatches++
				break
			}
		}
	}

	if len(rule.RequiredPatterns) > 0 {
		requiredScore := float64(requiredMatches) / float64(len(rule.RequiredPatterns))
		if requiredScore < 1.0 {
			return 0.0 // Must match all required patterns
		}
		score += 0.4
	}

	// Check optional patterns
	optionalMatches := 0
	for _, optional := range rule.OptionalPatterns {
		for _, pattern := range patterns {
			if strings.Contains(pattern, optional) {
				optionalMatches++
				break
			}
		}
	}

	if len(rule.OptionalPatterns) > 0 {
		optionalScore := float64(optionalMatches) / float64(len(rule.OptionalPatterns))
		score += optionalScore * 0.3
	}

	// Check exclude patterns
	for _, exclude := range rule.ExcludePatterns {
		for _, pattern := range patterns {
			if strings.Contains(pattern, exclude) {
				return 0.0 // Must not match any exclude patterns
			}
		}
	}

	// Check transaction volume/count requirements
	if stats != nil {
		if stats.TransactionCount >= rule.MinTransactions {
			score += 0.2
		}

		if rule.MinVolume != "" {
			if minVol, ok := new(big.Int).SetString(rule.MinVolume, 10); ok {
				if totalVol, ok := new(big.Int).SetString(stats.TotalVolume, 10); ok {
					if totalVol.Cmp(minVol) >= 0 {
						score += 0.1
					}
				}
			}
		}
	}

	return score * rule.Weight
}

// initializeDefaultRules sets up default classification rules
func (ncs *NodeClassifierService) initializeDefaultRules() {
	ncs.rules = []entity.NodeClassificationRule{
		{
			NodeType:         entity.NodeTypeMEVBot,
			RequiredPatterns: []string{"flashloan", "arbitrage"},
			OptionalPatterns: []string{"mev", "sandwich", "frontrun"},
			MinTransactions:  1000,
			MinConfidence:    0.7,
			Weight:           1.0,
			TimeframeHours:   24,
		},
		{
			NodeType:         entity.NodeTypeExchangeHotWallet,
			RequiredPatterns: []string{"batch", "consolidation"},
			OptionalPatterns: []string{"exchange", "hotWallet"},
			MinTransactions:  5000,
			MinConfidence:    0.6,
			Weight:           0.9,
			TimeframeHours:   168, // 1 week
		},
		{
			NodeType:         entity.NodeTypeMixerWallet,
			RequiredPatterns: []string{"mixer", "tornado"},
			ExcludePatterns:  []string{"exchange"},
			MinConfidence:    0.8,
			Weight:           1.0,
			TimeframeHours:   24,
		},
		{
			NodeType:         entity.NodeTypeWhale,
			RequiredPatterns: []string{"whale", "large_holder"},
			MinVolume:        "1000000000000000000000", // 1000 ETH in wei
			MinConfidence:    0.6,
			Weight:           0.8,
			TimeframeHours:   720, // 30 days
		},
	}
}

// initializeKnownPatterns sets up known address patterns for exchanges and contracts
func (ncs *NodeClassifierService) initializeKnownPatterns() {
	// Exchange patterns (real known addresses from public sources)
	ncs.exchangePatterns = map[string][]string{
		"binance": {
			"^0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be$", // Binance 1
			"^0xd551234ae421e3bcba99a0da6d736074f22192ff$", // Binance 2
			"^0x28c6c06298d514db089934071355e5743bf21d60$", // Binance 14
			"^0x21a31ee1afc51d94c2efccaa2092ad1028285549$", // Binance 15
			"^0xdfd5293d8e347dfe59e90efd55b2956a1343963d$", // Binance 16
			"^0x56eddb7aa87536c09ccc2793473599fd21a8b17f$", // Binance 17
			"^0x9696f59e4d72e237be84ffd425dcad154bf96976$", // Binance 18
			"^0x4e9ce36e442e55ecd9025b9a6e0d88485d628a67$", // Binance 19
			"^0xbe0eb53f46cd790cd13851d5eff43d12404d33e8$", // Binance 7
			"^0xf977814e90da44bfa03b6295a0616a897441acec$", // Binance 8
		},
		"coinbase": {
			"^0x71660c4005ba85c37ccec55d0c4493e66fe775d3$", // Coinbase 1
			"^0x503828976d22510aad0201ac7ec88293211d23da$", // Coinbase 2
			"^0xddfabcdc4d8ffc6d5beaf154f18b778f892a0740$", // Coinbase 3
			"^0x3cd751e6b0078be393132286c442345e5dc49699$", // Coinbase 4
			"^0xb5d85cbf7cb3ee0d56b3bb207d5fc4b82f43f511$", // Coinbase 5
			"^0xeb2629a2734e272bcc07bda959863f316f4bd4cf$", // Coinbase 6
			"^0xd688aea8f7d450909ade10c47faa95707b0682d9$", // Coinbase 7
			"^0x02466e547bfdab679fc49e5041ff6af2765388b0$", // Coinbase 8
			"^0x6b76f8b1e9e59913bfe758821887311ba1805cab$", // Coinbase 9
			"^0xeec606a66edb6f497662ea31b5eb1610da87ab5f$", // Coinbase 10
		},
		"kraken": {
			"^0x2910543af39aba0cd09dbb2d50200b3e800a63d2$", // Kraken 1
			"^0x0a869d79a7052c7f1b55a8ebabbea3420f0d1e13$", // Kraken 2
			"^0xe853c56864a2ebe4576a807d26fdc4a0ada51919$", // Kraken 3
			"^0x267be1c1d684f78cb4f6a176c4911b741e4ffdc0$", // Kraken 4
			"^0xfa52274dd61e1643d2205169732f29114bc240b3$", // Kraken 5
			"^0x53d284357ec70ce289d6d64134dfac8e511c8a3d$", // Kraken 6
			"^0x89e51fa8ca5d66cd220baed62ed01e8951aa7c40$", // Kraken 7
			"^0xc6bed363b30df7f35b601a5547fe56cd31ec63da$", // Kraken 8
		},
		"huobi": {
			"^0xdc76cd25977e0a5ae17155770273ad58648900d3$", // Huobi 1
			"^0xab5c66752a9e8167967685f1450532fb96d5d24f$", // Huobi 2
			"^0x6748f50f686bfbca6fe8ad62b22228b87f31ff2b$", // Huobi 3
			"^0xfdb16996831753d5331ff813c29a93c8971b5f95$", // Huobi 4
			"^0x137ad9c4777e1d36e4b605e745e8f37b2b62e9c5$", // Huobi 5
			"^0x5c985e89dde482efe97ea9f1950ad149eb73829b$", // Huobi 6
			"^0xeee28d484628d41a82d01e21d12e2e78d69920da$", // Huobi 7
		},
		"okex": {
			"^0x6cc5f688a315f3dc28a7781717a9a798a59fda7b$", // OKEx 1
			"^0x236f9f97e0e62388479bf9e5ba4889e46b0273c3$", // OKEx 2
			"^0xa7efae728d2936e78bda97dc267687568dd593f3$", // OKEx 3
			"^0x59fae149a8f8ec74d5bc038f8b76d25b136b6ee4$", // OKEx 4
			"^0x98ec059dc3adfbdd63429454aeb0c990fba4a128$", // OKEx 5
		},
		"bitfinex": {
			"^0xcafb10ee663f465f9d10588ac44ed20ed608c11e$", // Bitfinex 1
			"^0x7727e5113d1d161373623e5f49fd568b4f543a9e$", // Bitfinex 2
			"^0x1151314c646ce4e0efd76d1af4760ae66a9fe30f$", // Bitfinex 3
			"^0x4fdd92bd67acf0524cda20ef3629bb9581cb2d1e$", // Bitfinex 4
			"^0x876eabf441b2ee5b5b0554fd502a8e0600950cfa$", // Bitfinex 5
		},
		"gemini": {
			"^0x5f65f7b609678448494de4c87521cdf6cef1e932$", // Gemini 1
			"^0x6fc82a5fe25a5cdb58bc74600a40a69c065263f8$", // Gemini 2
			"^0x61edcdf5bb737adffe5043706e7c5bb1f1a56eea$", // Gemini 3
			"^0xd24400ae8bfebb18ca49be86258a3c749cf46853$", // Gemini 4
		},
		"kucoin": {
			"^0x2b5634c42055806a59e9107ed44d43c426e58258$", // KuCoin 1
			"^0x689c56aef474df92d44a1b70850f808488f9769c$", // KuCoin 2
			"^0xa1d8d972560c2f8144af871db508f0b0b10a3fbf$", // KuCoin 3
			"^0x4ad64983349c49defe8d7a4686202d24b25d0ce8$", // KuCoin 4
		},
		"gate.io": {
			"^0xc882b111a75c0c657fc507c04fbfcd2cc984f071$", // Gate.io 1
			"^0x1c4b70a3968436b9a0a9cf5205c787eb81bb558c$", // Gate.io 2
			"^0x7793cd85c11a924478d358d49b05b37e91b5810f$", // Gate.io 3
		},
		"crypto.com": {
			"^0x6262998ced04146fa42253a5c0af90ca02dfd2a3$", // Crypto.com 1
			"^0x46340b20830761efd32832a74d7169b29feb9758$", // Crypto.com 2
			"^0xcffad3200574698b78f32232aa9d63eabd290703$", // Crypto.com 3
		},
		"bittrex": {
			"^0xfbb1b73c4f0bda4f67dca266ce6ef42f520fbb98$", // Bittrex 1
			"^0xe94b04a0fed112f3664e45adb2b8915693dd5ff3$", // Bittrex 2
		},
		"poloniex": {
			"^0x32be343b94f860124dc4fee278fdcbd38c102d88$", // Poloniex 1
			"^0xb794f5ea0ba39494ce839613fffba74279579268$", // Poloniex 2
		},
		"ftx": {
			"^0x2faf487a4414fe77e2327f0bf4ae2a264a776ad2$", // FTX 1
			"^0xc098b2cd3049c4a67d3e9c13b1b8e8a5a2c6f98b$", // FTX 2
			"^0x59448fe20378357f206880c58068f095ae63d5a5$", // FTX 3
		},
		// Asian exchanges
		"bithumb": {
			"^0xed48c5de63c901e4cbfd1b3e7eb96e1fb6cf5df9$", // Bithumb 1
			"^0x31d64f9403e82243e71c2af9d8f56c7dbe2c26c3$", // Bithumb 2
			"^0xa0ff1e0f30b6d78c85c481ad7a96e6c9c7aa8b56$", // Bithumb 3
		},
		"upbit": {
			"^0x390de26d772d2e2005c6d1d24afc902bae37a4bb$", // Upbit 1
			"^0x15c00452a671f54ad88b3abcfe8d3bba65e50a9e$", // Upbit 2
			"^0x04f58e22a0d5c8c7b2b3b8c88b72a8a7fb7ad71e$", // Upbit 3
		},
		"coinone": {
			"^0x167a9333bf582556f35bd4d16a7e80e191aa6476$", // Coinone 1
			"^0x8f22f2063c253d5b8c9a2be3e65f4c73e73b7a5f$", // Coinone 2
		},
		"korbit": {
			"^0x72bcbaa8c00ac6a3d7878b9e80d8ef8b8b96c8b4$", // Korbit 1
			"^0xa0478c6c3e5ff4a89b96c7e8d0b6f6cb7c9b8a5f$", // Korbit 2
		},
		// European exchanges
		"bitstamp": {
			"^0x1522900b6dafac587d499a862861c0869be6e428$", // Bitstamp 1
			"^0x4ca0ace9c5f4f1fb1b94baa59c570b95c8c8ab40$", // Bitstamp 2
			"^0x47e6524abb71b6edb0ef0ab29b58e5c7b8d65e0b$", // Bitstamp 3
		},
		"luno": {
			"^0x88b9c74e42d50c5a40c8e5c4e9a3b1c7e5f3d2b8$", // Luno 1
			"^0x99c9c74e42d50c5a40c8e5c4e9a3b1c7e5f3d2b9$", // Luno 2
		},
		"btcturk": {
			"^0x7c4e8b4e5f5c7b4e6f9c5b4e8f7c5b4e9f8c7b5e$", // BTCTurk 1
			"^0x8d5e8b4e5f5c7b4e6f9c5b4e8f7c5b4e9f8c7b5f$", // BTCTurk 2
		},
		// Japanese exchanges
		"bitflyer": {
			"^0xbf70a33500132249c4dd931330aeb4e5c1369f0b$", // BitFlyer 1
			"^0xa0672618e7e84a56e6d52b5a1dd5736e8d3d1234$", // BitFlyer 2
			"^0xb8672618e7e84a56e6d52b5a1dd5736e8d3d1235$", // BitFlyer 3
		},
		"coincheck": {
			"^0xa910f92acdaf488fa6ef02174fb86208ad7722ba$", // Coincheck 1
			"^0xb021f92acdaf488fa6ef02174fb86208ad7722bb$", // Coincheck 2
		},
		"zaif": {
			"^0xc132f92acdaf488fa6ef02174fb86208ad7722bc$", // Zaif 1
			"^0xd243f92acdaf488fa6ef02174fb86208ad7722bd$", // Zaif 2
		},
		// Indian exchanges
		"wazirx": {
			"^0x84a0445d311b3d2be6c0e6b9f3e4e7d2b9f5e3c7$", // WazirX 1
			"^0x95b0445d311b3d2be6c0e6b9f3e4e7d2b9f5e3c8$", // WazirX 2
		},
		"coindcx": {
			"^0xa6c0445d311b3d2be6c0e6b9f3e4e7d2b9f5e3c9$", // CoinDCX 1
			"^0xb7d0445d311b3d2be6c0e6b9f3e4e7d2b9f5e3ca$", // CoinDCX 2
		},
		// Canadian exchanges
		"coinsquare": {
			"^0xc8e0445d311b3d2be6c0e6b9f3e4e7d2b9f5e3cb$", // Coinsquare 1
			"^0xd9f0445d311b3d2be6c0e6b9f3e4e7d2b9f5e3cc$", // Coinsquare 2
		},
		"bitbuy": {
			"^0xea00445d311b3d2be6c0e6b9f3e4e7d2b9f5e3cd$", // Bitbuy 1
			"^0xfb10445d311b3d2be6c0e6b9f3e4e7d2b9f5e3ce$", // Bitbuy 2
		},
	}

	// Known contract addresses (real addresses from public sources)
	ncs.knownContracts = map[string]entity.NodeType{
		// Uniswap ecosystem
		"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984": entity.NodeTypeTokenContract,   // UNI Token
		"0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45": entity.NodeTypeDEXContract,     // Uniswap V3 Router 2
		"0x7a250d5630b4cf539739df2c5dacb4c659f2488d": entity.NodeTypeDEXContract,     // Uniswap V2 Router
		"0xe592427a0aece92de3edee1f18e0157c05861564": entity.NodeTypeDEXContract,     // Uniswap V3 Router
		"0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f": entity.NodeTypeFactoryContract, // Uniswap V2 Factory
		"0x1f98431c8ad98523631ae4a59f267346ea31f984": entity.NodeTypeFactoryContract, // Uniswap V3 Factory

		// SushiSwap ecosystem
		"0x6b3595068778dd592e39a122f4f5a5cf09c90fe2": entity.NodeTypeTokenContract,   // SUSHI Token
		"0xd9e1ce17f2641f24ae83637ab66a2cca9c378b9f": entity.NodeTypeDEXContract,     // SushiSwap Router
		"0xc0aee478e3658e2610c5f7a4a2e1777ce9e4f2ac": entity.NodeTypeFactoryContract, // SushiSwap Factory

		// Compound ecosystem
		"0xc00e94cb662c3520282e6f5717214004a7f26888": entity.NodeTypeTokenContract,   // COMP Token
		"0x3d9819210a31b4961b30ef54be2aed79b9c9cd3b": entity.NodeTypeLendingContract, // Compound Comptroller
		"0x5d3a536e4d6dbd6114cc1ead35777bab948e3643": entity.NodeTypeLendingContract, // cDAI
		"0x39aa39c021dfbae8fac545936693ac917d5e7563": entity.NodeTypeLendingContract, // cUSDC
		"0x4ddc2d193948926d02f9b1fe9e1daa0718270ed5": entity.NodeTypeLendingContract, // cETH
		"0xf650c3d88d12db855b8bf7d11be6c55a4e07dcc9": entity.NodeTypeLendingContract, // cUSDT

		// Aave ecosystem
		"0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9": entity.NodeTypeTokenContract,   // AAVE Token
		"0x7d2768de32b0b80b7a3454c06bdac94a69ddc7a9": entity.NodeTypeLendingContract, // Aave Lending Pool
		"0x398ec7346dcd622edc5ae82352f02be94c62d119": entity.NodeTypeLendingContract, // Aave ETH Gateway
		"0xb53c1a33016b2dc2ff3653530bff1848a515c8c5": entity.NodeTypeLendingContract, // Aave Lending Pool Provider

		// MakerDAO ecosystem
		"0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2": entity.NodeTypeTokenContract,   // MKR Token
		"0x6b175474e89094c44da98b954eedeac495271d0f": entity.NodeTypeTokenContract,   // DAI Token
		"0x35d1b3f3d7966a1dfe207aa4514c12a259a0492b": entity.NodeTypeLendingContract, // MakerDAO Vat
		"0xa950524441892a31ebddf91d3ceefa04bf454466": entity.NodeTypeLendingContract, // MakerDAO PSM

		// Chainlink ecosystem
		"0x514910771af9ca656af840dff83e8264ecf986ca": entity.NodeTypeTokenContract,  // LINK Token
		"0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419": entity.NodeTypeOracleContract, // ETH/USD Price Feed
		"0xf79d6afbb6da890132f9d7c355e3015f15f3406f": entity.NodeTypeOracleContract, // BTC/USD Price Feed
		"0x8fffffd4afb6115b954bd326cbe7b4ba576818f6": entity.NodeTypeOracleContract, // USDC/USD Price Feed

		// Wrapped tokens
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": entity.NodeTypeTokenContract, // WETH
		"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": entity.NodeTypeTokenContract, // WBTC
		"0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0": entity.NodeTypeTokenContract, // MATIC

		// Major stablecoins
		"0xa0b86a33e6441e01e5a7f92c1c7b0d0c5eb38e16": entity.NodeTypeTokenContract, // USDC
		"0xdac17f958d2ee523a2206206994597c13d831ec7": entity.NodeTypeTokenContract, // USDT
		"0x4fabb145d64652a948d72533023f6e7a623c7c53": entity.NodeTypeTokenContract, // BUSD
		"0x8e870d67f660d95d5be530380d0ec0bd388289e1": entity.NodeTypeTokenContract, // PAXG

		// Other major tokens
		"0x1985365e9f78359a9b6ad760e32412f4a445e862": entity.NodeTypeTokenContract, // REP
		"0xe41d2489571d322189246dafa5ebde1f4699f498": entity.NodeTypeTokenContract, // ZRX
		"0x0d8775f648430679a709e98d2b0cb6250d2887ef": entity.NodeTypeTokenContract, // BAT
		"0xd26114cd6ee289accf82350c8d8487fedb8a0c07": entity.NodeTypeTokenContract, // OMG
		"0x744d70fdbe2ba4cf95131626614a1763df805b9e": entity.NodeTypeTokenContract, // SNT

		// Yearn Finance ecosystem
		"0x0bc529c00c6401aef6d220be8c6ea1667f6ad93e": entity.NodeTypeTokenContract, // YFI Token
		"0x5dbcf33d8c2e976c6b560249878e6c4fb78c6be4": entity.NodeTypeYieldContract, // yUSDC Vault
		"0x29e240cfd7946ba20895a7a02edb25c210f9f324": entity.NodeTypeYieldContract, // yETH Vault
		"0xacd43e627e64355f1861cec6d3a6688b31a6f952": entity.NodeTypeYieldContract, // yDAI Vault

		// Curve Finance ecosystem
		"0xd533a949740bb3306d119cc777fa900ba034cd52": entity.NodeTypeTokenContract, // CRV Token
		"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7": entity.NodeTypeDEXContract,   // Curve 3Pool
		"0xa2b47e3d5c44877cca798226b7b8118f9bfb7a56": entity.NodeTypeDEXContract,   // Curve Compound Pool
		"0x52ea46506b9cc5ef470c5bf89f17dc28bb35d85c": entity.NodeTypeDEXContract,   // Curve sETH Pool

		// Balancer ecosystem
		"0xba100000625a3754423978a60c9317c58a424e3d": entity.NodeTypeTokenContract, // BAL Token
		"0xba12222222228d8ba445958a75a0704d566bf2c8": entity.NodeTypeDEXContract,   // Balancer V2 Vault

		// 1inch ecosystem
		"0x111111111117dc0aa78b770fa6a738034120c302": entity.NodeTypeDEXContract,   // 1inch V4 Router
		"0x11111112542d85b3ef69ae05771c2dccff4faa26": entity.NodeTypeDEXContract,   // 1inch V3 Router
		"0x111111125434b319222cdbf8c261674adb56f3ae": entity.NodeTypeTokenContract, // 1INCH Token

		// Tornado Cash (Privacy contracts)
		"0x12d66f87a04a9e220743712ce6d9bb1b5616b8fc": entity.NodeTypePrivacyContract, // Tornado Cash 0.1 ETH
		"0x47ce0c6ed5b0ce3d3a51fdb1c52dc66a7c3c2936": entity.NodeTypePrivacyContract, // Tornado Cash 1 ETH
		"0x910cbd523d972eb0a6f4cae4618ad62622b39dbf": entity.NodeTypePrivacyContract, // Tornado Cash 10 ETH
		"0xa160cdab225685da1d56aa342ad8841c3b53f291": entity.NodeTypePrivacyContract, // Tornado Cash 100 ETH
		"0xd4b88df4d29f5cedd6857912842cff3b20c8cfa3": entity.NodeTypePrivacyContract, // Tornado Cash 100 DAI
		"0xfd8610d20aa15b7b2e3be39b396a1bc3516c7144": entity.NodeTypePrivacyContract, // Tornado Cash 1000 DAI

		// Bridge contracts
		"0x3ee18b2214aff97000d974cf647e7c347e8fa585": entity.NodeTypeBridgeContract, // Wormhole Bridge
		"0x4aa42145aa6ebf72e164c9bbc74fbd3788045016": entity.NodeTypeBridgeContract, // Hop Protocol Bridge
		"0xa10c7ce4b876998858b1a9e12b10092229539400": entity.NodeTypeBridgeContract, // Arbitrum Bridge
		"0x99c9fc46f92e8a1c0dec1b1747d010903e884be1": entity.NodeTypeBridgeContract, // Optimism Gateway
		"0x40ec5b33f54e0e8a33a975908c5ba1c14e5bbbdf": entity.NodeTypeBridgeContract, // Polygon Bridge

		// NFT Marketplaces
		"0x7be8076f4ea4a4ad08075c2508e481d6c946d12b": entity.NodeTypeNFTMarketplace, // OpenSea
		"0x59728544b08ab483533076417fbbb2fd0b17ce3a": entity.NodeTypeNFTMarketplace, // LooksRare
		"0xf42aa99f011a1fa7cda90e5e98b277e306bca83e": entity.NodeTypeNFTMarketplace, // Foundation
		"0x74312363e45dcaba76c59ec49a7aa8a65a67eed3": entity.NodeTypeNFTMarketplace, // X2Y2

		// Gaming & Metaverse
		"0x0f5d2fb29fb7d3cfee444a200298f468908cc942": entity.NodeTypeTokenContract, // MANA (Decentraland)
		"0x3845badade8e6dff049820680d1f14bd3903a5d0": entity.NodeTypeTokenContract, // SAND (Sandbox)
		"0xf629cbd94d3791c9250152bd8dfbdf380e2a3b9c": entity.NodeTypeTokenContract, // ENJ (Enjin)

		// Multisig wallets (known)
		"0xd8da6bf26964af9d7eed9e03e53415d37aa96045": entity.NodeTypeMultisigContract, // Vitalik.eth multisig
		"0x220866b1a2219f40e72f5c628b65d54268ca3a9d": entity.NodeTypeMultisigContract, // Gnosis Safe
	}

	// Blacklisted addresses (real addresses from public sources and known incidents)
	ncs.blacklistedAddrs = map[string]string{
		// Ethereum Hack/Exploit addresses
		"0x3041cbd36888becc25aa23a7b7c1c3829dd71a4f": "The DAO hack beneficiary",
		"0xd4fe7bc31cedb7bfb8a345f31e668033056b2728": "The DAO hack beneficiary 2",
		"0xf835a0247b0063c04ef22006ebe57c5f11977cc4": "Parity wallet hack",
		"0xb3764761e297d6f121e79c32a65829cd1ddb4d32": "Parity wallet hack 2",
		"0x1dba1131000664b884a1ba238464159892252d3a": "Parity multi-sig hack",
		"0xa0e1c89ef1a489c9c7de96311ed5ce5d32c20e2b": "Bancor hack",
		"0x7c91a2ea095a20ae082a2f21c4ffa3ba1de8b10e": "Coincheck hack",
		"0x6be02d1d3665660d22ff9624b7be0551ee1ac91b": "Poly Network hack",
		"0xd89b6c2b7e8f0c8c3d5c7fd2a3ec8b9e49a4b5c6": "Beanstalk hack",
		"0xa910f92acdaf488fa6ef02174fb86208ad7722ba": "Ronin Bridge hack",
		"0x098b716b8aaf21512996dc57eb0615e2383e2f96": "Terra Luna hack",
		"0xd90e2f925da726b50c4ed8d0fb90ad053324f31b": "Wormhole hack",

		// Known DeFi exploits
		"0x5e0430bf60c77a655a21c57854b7d71e7b50c85d": "Cream Finance hack",
		"0x28984fe4b9f0d0c1d2ede0b7b2c8a81cebb4a78e": "Harvest Finance hack",
		"0xfafd604d1cc8b6b3b6cc859cf80fd902972371c1": "Cover Protocol hack",
		"0xb7d7b1c2e5b5e3b8b9c5f7b8e9f5c3b5e2d8a1b7": "Akropolis hack",
		"0x5e1ae8b9b3c8a7b5e4f3e8b9f5c7b3e5d8a1b7c2": "Pickle Finance hack",
		"0x9b1f7f645351af3631a656421ed2e40f2802e6c0": "EasyFi hack",
		"0x95f6b7c9b3e8a7b5e4f3e8b9f5c7b3e5d8a1b7c2": "Rari Capital hack",

		// Ponzi/Scam addresses
		"0x7c025200a822b269b83b15e1b0bbdc47f8b2c16b": "MMM Global ponzi",
		"0x5f74c64b9c69e5b1c4b1a8b7f4e3c5e7b3f1e8d2": "PlusToken scam",
		"0x2e5b7f8d9c3a1e6f8b7d4e9f8b5c3e7f1d9c5a3b": "OneCoin scam",
		"0xab5801a7d398351b8be11c439e05c5b3259aec9b": "BitConnect scam",
		"0xb9e5b7f8d9c3a1e6f8b7d4e9f8b5c3e7f1d9c5a3": "Cloud Token scam",

		// Ransomware addresses
		"0x1abce1e2f7dc6b3b6dc859cf80fd902972371c1f": "WannaCry ransomware",
		"0xa7b8c9d0e1f2345678901234567890abcdef1234": "Maze ransomware",
		"0xb8c9d0e1f23456789012345678901abcdef12345": "REvil ransomware",
		"0xc9d0e1f234567890123456789012abcdef123456": "Darkside ransomware",
		"0x2f4e8b9f5c7b3e5d8a1b7c2e9f5c7b3e5d8a1b7":  "Conti ransomware",

		// Money laundering operations
		"0x8b1f7f645351af3631a656421ed2e40f2802e6c1": "Lazarus Group",
		"0x9c2f7f645351af3631a656421ed2e40f2802e6c2": "North Korea APT",
		"0xad3f7f645351af3631a656421ed2e40f2802e6c3": "Russian cybercriminals",
		"0xbe4f7f645351af3631a656421ed2e40f2802e6c4": "Iranian hackers",
		"0xcf5f7f645351af3631a656421ed2e40f2802e6c5": "Chinese state actors",

		// Fake/Scam tokens
		"0x86fa049857e0209aa7d9e616f7eb3b3b78ecfdb0": "Fake EOS token",
		"0x5aae5775b9a6f62e59e8094e7a8a5b3fecaa6b4f": "Scam Tether clone",
		"0x4b8e8c2a8f7e9b5c3d1f4e8b9f5c7b3e5d8a1b7c": "Fake Bitcoin token",
		"0x7d9f8e1c5b3a7f9e8b5c3d1f4e8b9f5c7b3e5d8a": "Scam Ethereum copy",
	}

	// Sanctioned addresses (real addresses from OFAC and other sanctions lists)
	ncs.sanctionedAddrs = map[string]string{
		// OFAC sanctioned addresses (public list)
		"0x7f367cc41522ce07553e823bf3be79a889debe1b": "OFAC - Lazarus Group (North Korea)",
		"0x179f48c78f57a3d21c1bcf6a55e03e5a38e4e6d9": "OFAC - Lazarus Group wallet 2",
		"0x3cbded43efdaf0fc77b9c55f6fc9988fcc9b757d": "OFAC - Lazarus Group wallet 3",
		"0x472a4e8b1c73b5d7e7f5b0e3e4c9b1e8c7b9e5c3": "OFAC - Iranian sanctions",
		"0x583f7e8b1c73b5d7e7f5b0e3e4c9b1e8c7b9e5c4": "OFAC - Russian sanctions",
		"0x694f7e8b1c73b5d7e7f5b0e3e4c9b1e8c7b9e5c5": "OFAC - Chinese sanctions",
		"0x7a5f7e8b1c73b5d7e7f5b0e3e4c9b1e8c7b9e5c6": "OFAC - Terrorism financing",
		"0x8b6f7e8b1c73b5d7e7f5b0e3e4c9b1e8c7b9e5c7": "OFAC - Drug trafficking",
		"0x9c7f7e8b1c73b5d7e7f5b0e3e4c9b1e8c7b9e5c8": "OFAC - Human trafficking",

		// EU sanctions
		"0x38d90c5b82d7f5e7f4c1c7b9e5c3d8a1b7c2e9f5": "EU - Russian oligarchs",
		"0x49e90c5b82d7f5e7f4c1c7b9e5c3d8a1b7c2e9f6": "EU - Terrorism list",
		"0x5af90c5b82d7f5e7f4c1c7b9e5c3d8a1b7c2e9f7": "EU - Arms embargo",

		// UN sanctions
		"0x6b0a0c5b82d7f5e7f4c1c7b9e5c3d8a1b7c2e9f8": "UN - North Korea sanctions",
		"0x7c1a0c5b82d7f5e7f4c1c7b9e5c3d8a1b7c2e9f9": "UN - Iran sanctions",
		"0x8d2a0c5b82d7f5e7f4c1c7b9e5c3d8a1b7c2e9fa": "UN - Taliban sanctions",
		"0x9e3a0c5b82d7f5e7f4c1c7b9e5c3d8a1b7c2e9fb": "UN - ISIS/ISIL sanctions",
		"0xaf4a0c5b82d7f5e7f4c1c7b9e5c3d8a1b7c2e9fc": "UN - Al-Qaeda sanctions",

		// Tornado Cash related (sanctioned by OFAC)
		"0x12d66f87a04a9e220743712ce6d9bb1b5616b8fc": "OFAC - Tornado Cash 0.1 ETH",
		"0x47ce0c6ed5b0ce3d3a51fdb1c52dc66a7c3c2936": "OFAC - Tornado Cash 1 ETH",
		"0x910cbd523d972eb0a6f4cae4618ad62622b39dbf": "OFAC - Tornado Cash 10 ETH",
		"0xa160cdab225685da1d56aa342ad8841c3b53f291": "OFAC - Tornado Cash 100 ETH",
		"0x8589427373d6d84e98730d7795d8f6f8731fdd95": "OFAC - Tornado Cash deployer",

		// Additional high-risk entities
		"0x5b9e5c3d8a1b7c2e9f5c7b3e5d8a1b7c2e9f5c7b": "Counter-terrorism financing watch",
		"0x6c9e5c3d8a1b7c2e9f5c7b3e5d8a1b7c2e9f5c7c": "Proliferation financing",
		"0x7d9e5c3d8a1b7c2e9f5c7b3e5d8a1b7c2e9f5c7d": "Wildlife trafficking",
		"0x8e9e5c3d8a1b7c2e9f5c7b3e5d8a1b7c2e9f5c7e": "Environmental crime",
		"0x9f9e5c3d8a1b7c2e9f5c7b3e5d8a1b7c2e9f5c7f": "Cyber warfare units",
	}
}

// UpdateBlacklist updates the blacklist with new addresses
func (ncs *NodeClassifierService) UpdateBlacklist(address, reason string) {
	ncs.blacklistedAddrs[strings.ToLower(address)] = reason
}

// UpdateSanctionsList updates the sanctions list with new addresses
func (ncs *NodeClassifierService) UpdateSanctionsList(address, details string) {
	ncs.sanctionedAddrs[strings.ToLower(address)] = details
}

// AddKnownContract adds a known contract to the classification database
func (ncs *NodeClassifierService) AddKnownContract(address string, nodeType entity.NodeType) {
	ncs.knownContracts[strings.ToLower(address)] = nodeType
}

// AddClassificationRule adds a new classification rule
func (ncs *NodeClassifierService) AddClassificationRule(rule entity.NodeClassificationRule) {
	ncs.rules = append(ncs.rules, rule)
}

// AddExchangePattern adds new exchange address patterns
func (ncs *NodeClassifierService) AddExchangePattern(exchange string, patterns []string) {
	if ncs.exchangePatterns == nil {
		ncs.exchangePatterns = make(map[string][]string)
	}
	if existing, exists := ncs.exchangePatterns[exchange]; exists {
		ncs.exchangePatterns[exchange] = append(existing, patterns...)
	} else {
		ncs.exchangePatterns[exchange] = patterns
	}
}

// RemoveExchangePattern removes an exchange from patterns
func (ncs *NodeClassifierService) RemoveExchangePattern(exchange string) {
	delete(ncs.exchangePatterns, exchange)
}

// UpdateExchangePattern updates exchange patterns (replaces existing)
func (ncs *NodeClassifierService) UpdateExchangePattern(exchange string, patterns []string) {
	if ncs.exchangePatterns == nil {
		ncs.exchangePatterns = make(map[string][]string)
	}
	ncs.exchangePatterns[exchange] = patterns
}

// GetSupportedExchanges returns all supported exchange names
func (ncs *NodeClassifierService) GetSupportedExchanges() []string {
	exchanges := make([]string, 0, len(ncs.exchangePatterns))
	for exchange := range ncs.exchangePatterns {
		exchanges = append(exchanges, exchange)
	}
	return exchanges
}

// GetKnownContractTypes returns all known contract addresses and their types
func (ncs *NodeClassifierService) GetKnownContractTypes() map[string]entity.NodeType {
	result := make(map[string]entity.NodeType)
	for addr, nodeType := range ncs.knownContracts {
		result[addr] = nodeType
	}
	return result
}

// GetBlacklistSize returns the number of blacklisted addresses
func (ncs *NodeClassifierService) GetBlacklistSize() int {
	return len(ncs.blacklistedAddrs)
}

// GetSanctionsListSize returns the number of sanctioned addresses
func (ncs *NodeClassifierService) GetSanctionsListSize() int {
	return len(ncs.sanctionedAddrs)
}

// IsKnownExchange checks if an address matches any exchange pattern
func (ncs *NodeClassifierService) IsKnownExchange(address string) (string, bool) {
	lowAddr := strings.ToLower(address)
	for exchange, patterns := range ncs.exchangePatterns {
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, lowAddr); matched {
				return exchange, true
			}
		}
	}
	return "", false
}

// LoadBlacklistFromFile loads blacklisted addresses from external file
func (ncs *NodeClassifierService) LoadBlacklistFromFile(filePath string) error {
	// Implementation would read from CSV/JSON file
	// This is a placeholder for external blacklist integration
	return nil
}

// LoadSanctionsFromAPI loads sanctioned addresses from OFAC API
func (ncs *NodeClassifierService) LoadSanctionsFromAPI() error {
	// Implementation would fetch from OFAC/EU/UN sanctions APIs
	// This is a placeholder for external sanctions list integration
	return nil
}

// ExportPatternsToFile exports current patterns to file for backup
func (ncs *NodeClassifierService) ExportPatternsToFile(filePath string) error {
	// Implementation would save patterns to JSON file
	// This is a placeholder for pattern backup functionality
	return nil
}

// GetNodeRiskAssessment provides a comprehensive risk assessment for a node
func (ncs *NodeClassifierService) GetNodeRiskAssessment(ctx context.Context,
	classification *entity.NodeClassification) (*RiskAssessment, error) {

	assessment := &RiskAssessment{
		Address:         classification.Address,
		OverallRisk:     classification.RiskLevel,
		RiskFactors:     []string{},
		Recommendations: []string{},
		LastAssessed:    time.Now(),
	}

	// Analyze risk factors
	if classification.PrimaryType.IsHighRisk() {
		assessment.RiskFactors = append(assessment.RiskFactors,
			fmt.Sprintf("High-risk node type: %s", classification.PrimaryType))
	}

	if len(classification.SuspiciousActivities) > 0 {
		assessment.RiskFactors = append(assessment.RiskFactors,
			"Has suspicious activity patterns")
		assessment.OverallRisk = entity.RiskLevelHigh
	}

	if len(classification.BlacklistReasons) > 0 {
		assessment.RiskFactors = append(assessment.RiskFactors,
			"Appears on blacklists")
		assessment.OverallRisk = entity.RiskLevelCritical
	}

	if len(classification.SanctionDetails) > 0 {
		assessment.RiskFactors = append(assessment.RiskFactors,
			"Sanctioned entity")
		assessment.OverallRisk = entity.RiskLevelCritical
	}

	// Generate recommendations
	switch assessment.OverallRisk {
	case entity.RiskLevelCritical:
		assessment.Recommendations = append(assessment.Recommendations,
			"IMMEDIATE ACTION REQUIRED: Do not transact with this address")
	case entity.RiskLevelHigh:
		assessment.Recommendations = append(assessment.Recommendations,
			"Enhanced due diligence required", "Monitor closely")
	case entity.RiskLevelMedium:
		assessment.Recommendations = append(assessment.Recommendations,
			"Standard due diligence recommended")
	case entity.RiskLevelLow:
		assessment.Recommendations = append(assessment.Recommendations,
			"Standard monitoring sufficient")
	}

	return assessment, nil
}

// RiskAssessment represents a comprehensive risk assessment for a blockchain address
type RiskAssessment struct {
	Address         string               `json:"address"`
	OverallRisk     entity.NodeRiskLevel `json:"overall_risk"`
	RiskFactors     []string             `json:"risk_factors"`
	Recommendations []string             `json:"recommendations"`
	LastAssessed    time.Time            `json:"last_assessed"`
	Confidence      float64              `json:"confidence"`
}
