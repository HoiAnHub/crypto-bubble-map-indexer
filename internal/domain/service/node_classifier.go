package service

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// BlockchainService interface for querying blockchain data
type BlockchainService interface {
	// GetCodeAt returns the contract bytecode at the given address
	GetCodeAt(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error)
}

// NodeClassifierService handles node classification logic
type NodeClassifierService struct {
	rules            []entity.NodeClassificationRule
	exchangePatterns map[string][]string // exchange name -> address patterns
	blacklistedAddrs map[string]string   // address -> reason
	sanctionedAddrs  map[string]string   // address -> sanction details
	knownContracts   map[string]entity.NodeType
	blockchain       BlockchainService // Add blockchain service
	logger           *logger.Logger    // Add logger
}

// NewNodeClassifierService creates a new node classifier service
func NewNodeClassifierService(blockchain BlockchainService, logger *logger.Logger) *NodeClassifierService {
	service := &NodeClassifierService{
		rules:            []entity.NodeClassificationRule{},
		exchangePatterns: make(map[string][]string),
		blacklistedAddrs: make(map[string]string),
		sanctionedAddrs:  make(map[string]string),
		knownContracts:   make(map[string]entity.NodeType),
		blockchain:       blockchain,
		logger:           logger.WithComponent("node-classifier"),
	}

	service.initializeDefaultRules()
	service.initializeKnownPatterns()

	return service
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

	// 2. CRITICAL IMPROVEMENT: Check if address is EOA or Contract using bytecode
	isContract, contractType, err := ncs.checkAddressType(ctx, address)
	if err != nil {
		ncs.logger.Warn("Failed to check address type",
			zap.String("address", address),
			zap.Error(err))
		// Continue with other classification methods if bytecode check fails
	} else {
		classification.DetectionMethods = append(classification.DetectionMethods, string(entity.DetectionMethodBytecodeAnalysis))

		if isContract {
			// Address is a smart contract
			classification.Tags = append(classification.Tags, "smart_contract")

			// Use contract type if detected
			if contractType != entity.NodeTypeUnknown {
				classification.PrimaryType = contractType
				classification.ConfidenceScore = 0.9
				classification.RiskLevel = contractType.GetDefaultRiskLevel()
				return classification, nil
			}
			// If contract type not determined, continue with further analysis
		} else {
			// Address is EOA (Externally Owned Account)
			classification.PrimaryType = entity.NodeTypeEOA
			classification.Tags = append(classification.Tags, "eoa")
			classification.ConfidenceScore = 0.9
			classification.RiskLevel = entity.RiskLevelLow
			// Continue with EOA-specific analysis
		}
	}

	// 3. Check known contracts
	if nodeType, exists := ncs.knownContracts[address]; exists {
		classification.PrimaryType = nodeType
		classification.RiskLevel = nodeType.GetDefaultRiskLevel()
		classification.ConfidenceScore = 0.9
		classification.DetectionMethods = append(classification.DetectionMethods, string(entity.DetectionMethodManual))
		return classification, nil
	}

	// 4. Check exchange patterns
	for exchange, patterns := range ncs.exchangePatterns {
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, address); matched {
				if isContract {
					classification.PrimaryType = entity.NodeTypeExchangeHotWallet // Exchange contract
				} else {
					classification.PrimaryType = entity.NodeTypeExchangeWallet // Exchange EOA
				}
				classification.Exchanges = []string{exchange}
				classification.ConfidenceScore = 0.8
				classification.RiskLevel = entity.RiskLevelLow
				classification.DetectionMethods = append(classification.DetectionMethods, string(entity.DetectionMethodPatternAnalysis))
				break
			}
		}
		if classification.PrimaryType != entity.NodeTypeUnknown {
			break
		}
	}

	// 5. Analyze transaction patterns if statistics available
	if stats != nil {
		ncs.analyzeTransactionPatterns(classification, stats, patterns, isContract)
	}

	// 6. Apply classification rules
	ncs.applyClassificationRules(classification, stats, patterns, isContract)

	// 7. Set default if still unknown
	if classification.PrimaryType == entity.NodeTypeUnknown {
		if isContract {
			// Default for unknown contracts
			classification.PrimaryType = entity.NodeTypeTokenContract // Most common contract type
			classification.RiskLevel = entity.RiskLevelLow
			classification.ConfidenceScore = 0.3
		} else {
			// Default for EOAs
			classification.PrimaryType = entity.NodeTypeEOA
			classification.RiskLevel = entity.RiskLevelLow
			classification.ConfidenceScore = 0.3
		}
		classification.DetectionMethods = append(classification.DetectionMethods, string(entity.DetectionMethodHeuristic))
	}

	return classification, nil
}

// analyzeTransactionPatterns analyzes transaction patterns to help classify nodes
func (ncs *NodeClassifierService) analyzeTransactionPatterns(classification *entity.NodeClassification,
	stats *entity.WalletStats, patterns []string, isContract bool) {

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
	stats *entity.WalletStats, patterns []string, isContract bool) {

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
	// Exchange patterns (simplified examples)
	ncs.exchangePatterns = map[string][]string{
		"binance":   {"^0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be$", "^0xd551234ae421e3bcba99a0da6d736074f22192ff$", "^0x48f9bE7AC3071B46853E8A1972ADE0b155Ce0333$"},
		"coinbase":  {"^0x71660c4005ba85c37ccec55d0c4493e66fe775d3$", "^0x503828976d22510aad0201ac7ec88293211d23da$"},
		"okex":      {"^0x6cc5f688a315f3dc28a7781717a9a798a59fda7b$"},
		"huobi":     {"^0xdc76cd25977e0a5ae17155770273ad58648900d3$", "^0xab5c66752a9e8167967685f1450532fb96d5d24f$"},
		"kraken":    {"^0x2910543af39aba0cd09dbb2d50200b3e800a63d2$", "^0x0a869d79a7052c7f1b55a8ebabbea3420f0d1e13$"},
		"ftx":       {"^0xc098b2cd3049c4a67d3e9c13b1b8e8a5a2c6f98b$"},
		"uniswap":   {"^0x1f9840a85d5af5bf1d1762f925bdaddc4201f984$", "^0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45$"},
		"sushiswap": {"^0x6b3595068778dd592e39a122f4f5a5cf09c90fe2$"},
	}

	// Known contract addresses (examples)
	ncs.knownContracts = map[string]entity.NodeType{
		"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984": entity.NodeTypeDEXContract,     // Uniswap Token
		"0x6b3595068778dd592e39a122f4f5a5cf09c90fe2": entity.NodeTypeDEXContract,     // SushiSwap Token
		"0xa0b86a33e6441e01e5a7f92c1c7b0d0c5eb38e16": entity.NodeTypePrivacyContract, // Tornado Cash (example)
		"0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0": entity.NodeTypeTokenContract,   // Polygon (MATIC)
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": entity.NodeTypeTokenContract,   // WETH
		"0xa0b73e1ff0b80914ab6fe0444e65848c4c34450b": entity.NodeTypeLendingContract, // Compound (example)
		"0x5d3a536e4d6dbd6114cc1ead35777bab948e3643": entity.NodeTypeLendingContract, // Compound cDAI
		"0x39aa39c021dfbae8fac545936693ac917d5e7563": entity.NodeTypeLendingContract, // Compound cUSDC
	}

	// Blacklisted addresses (examples - should be loaded from external sources)
	ncs.blacklistedAddrs = map[string]string{
		"0x1234567890abcdef1234567890abcdef12345678": "Known ransomware address",
		"0x9876543210fedcba9876543210fedcba98765432": "Ponzi scheme contract",
	}

	// Sanctioned addresses (examples - should be loaded from OFAC list)
	ncs.sanctionedAddrs = map[string]string{
		"0x1111111111111111111111111111111111111111": "OFAC Sanctions List",
		"0x2222222222222222222222222222222222222222": "UN Sanctions List",
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

// checkAddressType checks if an address is a contract or EOA by examining bytecode
func (ncs *NodeClassifierService) checkAddressType(ctx context.Context, address string) (isContract bool, contractType entity.NodeType, err error) {
	// Get bytecode at the address
	code, err := ncs.blockchain.GetCodeAt(ctx, address, nil) // nil = latest block
	if err != nil {
		return false, entity.NodeTypeUnknown, fmt.Errorf("failed to get code at address %s: %w", address, err)
	}

	// If bytecode is empty or just "0x", it's an EOA
	if len(code) == 0 {
		return false, entity.NodeTypeUnknown, nil
	}

	// If bytecode exists, it's a contract
	isContract = true

	// Try to determine contract type based on bytecode patterns
	contractType = ncs.analyzeContractType(code, address)

	return isContract, contractType, nil
}

// analyzeContractType attempts to determine contract type from bytecode
func (ncs *NodeClassifierService) analyzeContractType(bytecode []byte, address string) entity.NodeType {
	// Convert bytecode to hex string for analysis
	codeHex := fmt.Sprintf("%x", bytecode)

	// Check for common contract patterns

	// ERC20 Token patterns (look for transfer, approve function signatures)
	if strings.Contains(codeHex, "a9059cbb") && strings.Contains(codeHex, "095ea7b3") {
		return entity.NodeTypeTokenContract
	}

	// Uniswap V2 Router patterns
	if strings.Contains(codeHex, "7ff36ab5") || strings.Contains(codeHex, "18cbafe5") {
		return entity.NodeTypeDEXContract
	}

	// Multicall patterns
	if strings.Contains(codeHex, "ac9650d8") {
		return entity.NodeTypeMultisigContract // or could be a new NodeTypeMulticall
	}

	// Compound/Lending patterns (mint, redeem)
	if strings.Contains(codeHex, "a6afed95") || strings.Contains(codeHex, "852a12e3") {
		return entity.NodeTypeLendingContract
	}

	// WETH pattern (deposit/withdraw)
	if strings.Contains(codeHex, "d0e30db0") && strings.Contains(codeHex, "2e1a7d4d") {
		// Could be WETH or other deposit/withdraw contract
		return entity.NodeTypeTokenContract
	}

	// Proxy contract patterns (common proxy bytecode)
	if strings.Contains(codeHex, "360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc") {
		return entity.NodeTypeProxyContract
	}

	// Factory contract patterns
	if strings.Contains(codeHex, "5af43d82803e903d91602b57fd5bf3") { // CREATE2 bytecode pattern
		return entity.NodeTypeFactoryContract
	}

	// Default to unknown contract if no patterns match
	return entity.NodeTypeTokenContract // Most contracts are tokens
}
