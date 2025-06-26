package entity

import (
	"time"
)

// NodeType represents different types of blockchain entities
type NodeType string

const (
	// Address Node Types
	NodeTypeEOA                NodeType = "EOA"                  // Externally Owned Account (regular user wallet)
	NodeTypeExchangeWallet     NodeType = "EXCHANGE_WALLET"      // Centralized exchange wallet
	NodeTypeExchangeHotWallet  NodeType = "EXCHANGE_HOT_WALLET"  // Exchange hot wallet (frequent trading)
	NodeTypeExchangeColdWallet NodeType = "EXCHANGE_COLD_WALLET" // Exchange cold wallet (long-term storage)
	NodeTypeBridgeWallet       NodeType = "BRIDGE_WALLET"        // Cross-chain bridge wallet
	NodeTypeMixerWallet        NodeType = "MIXER_WALLET"         // Money laundering/privacy mixer
	NodeTypeMEVBot             NodeType = "MEV_BOT"              // Maximum Extractable Value bot
	NodeTypeArbitrageBot       NodeType = "ARBITRAGE_BOT"        // Price arbitrage bot
	NodeTypeMarketMaker        NodeType = "MARKET_MAKER"         // Market making bot/service
	NodeTypeWhale              NodeType = "WHALE"                // High net worth individual
	NodeTypeSuspiciousWallet   NodeType = "SUSPICIOUS_WALLET"    // Flagged for suspicious activity
	NodeTypeBlacklistedWallet  NodeType = "BLACKLISTED_WALLET"   // Officially blacklisted address

	// Contract Node Types
	NodeTypeDEXContract       NodeType = "DEX_CONTRACT"       // Decentralized exchange contract
	NodeTypeLendingContract   NodeType = "LENDING_CONTRACT"   // Lending protocol contract
	NodeTypeStakingContract   NodeType = "STAKING_CONTRACT"   // Staking contract
	NodeTypeNFTMarketplace    NodeType = "NFT_MARKETPLACE"    // NFT marketplace contract
	NodeTypeDAOContract       NodeType = "DAO_CONTRACT"       // DAO governance contract
	NodeTypeGamblingContract  NodeType = "GAMBLING_CONTRACT"  // Gambling/gaming contract
	NodeTypePonziContract     NodeType = "PONZI_CONTRACT"     // Ponzi scheme/scam contract
	NodeTypePrivacyContract   NodeType = "PRIVACY_CONTRACT"   // Privacy protocol (e.g., Tornado Cash)
	NodeTypeTokenContract     NodeType = "TOKEN_CONTRACT"     // ERC20/ERC721/ERC1155 token contract
	NodeTypeBridgeContract    NodeType = "BRIDGE_CONTRACT"    // Cross-chain bridge contract
	NodeTypeYieldContract     NodeType = "YIELD_CONTRACT"     // Yield farming contract
	NodeTypeInsuranceContract NodeType = "INSURANCE_CONTRACT" // DeFi insurance contract
	NodeTypeOracleContract    NodeType = "ORACLE_CONTRACT"    // Price/data oracle contract
	NodeTypeMultisigContract  NodeType = "MULTISIG_CONTRACT"  // Multi-signature wallet contract
	NodeTypeProxyContract     NodeType = "PROXY_CONTRACT"     // Proxy/upgradeable contract
	NodeTypeFactoryContract   NodeType = "FACTORY_CONTRACT"   // Contract factory

	// Service Node Types
	NodeTypeOracleService     NodeType = "ORACLE_SERVICE"     // Oracle data provider
	NodeTypeFlashLoanProvider NodeType = "FLASHLOAN_PROVIDER" // Flash loan service
	NodeTypeYieldAggregator   NodeType = "YIELD_AGGREGATOR"   // Yield farming aggregator
	NodeTypeLiquidityProvider NodeType = "LIQUIDITY_PROVIDER" // LP in various protocols
	NodeTypeValidator         NodeType = "VALIDATOR"          // PoS validator
	NodeTypeMiner             NodeType = "MINER"              // PoW miner

	// Exchange-specific Types
	NodeTypeCEXDeposit    NodeType = "CEX_DEPOSIT"    // CEX deposit address
	NodeTypeCEXWithdrawal NodeType = "CEX_WITHDRAWAL" // CEX withdrawal address
	NodeTypeCEXSettlement NodeType = "CEX_SETTLEMENT" // CEX settlement address

	// Special Categories
	NodeTypeDarkWeb            NodeType = "DARKWEB"             // Dark web related
	NodeTypeRansomware         NodeType = "RANSOMWARE"          // Ransomware related
	NodeTypeTerroristFinancing NodeType = "TERRORIST_FINANCING" // Terrorist financing
	NodeTypeMoneyLaundering    NodeType = "MONEY_LAUNDERING"    // Money laundering
	NodeTypeSanctioned         NodeType = "SANCTIONED"          // Government sanctioned

	// Unknown/Default
	NodeTypeUnknown NodeType = "UNKNOWN" // Unknown type
)

// NodeRiskLevel represents the risk level of a node
type NodeRiskLevel string

const (
	RiskLevelLow      NodeRiskLevel = "LOW"
	RiskLevelMedium   NodeRiskLevel = "MEDIUM"
	RiskLevelHigh     NodeRiskLevel = "HIGH"
	RiskLevelCritical NodeRiskLevel = "CRITICAL"
	RiskLevelUnknown  NodeRiskLevel = "UNKNOWN"
)

// NodeClassification represents comprehensive node classification
type NodeClassification struct {
	Address             string        `json:"address"`
	PrimaryType         NodeType      `json:"primary_type"`
	SecondaryTypes      []NodeType    `json:"secondary_types"`
	RiskLevel           NodeRiskLevel `json:"risk_level"`
	ConfidenceScore     float64       `json:"confidence_score"`  // 0.0 - 1.0
	DetectionMethods    []string      `json:"detection_methods"` // ["pattern_analysis", "ml_model", "manual"]
	Tags                []string      `json:"tags"`              // ["high_volume", "whale", "bot"]
	Exchanges           []string      `json:"exchanges"`         // Associated exchanges if any
	Protocols           []string      `json:"protocols"`         // Associated DeFi protocols
	LastClassified      time.Time     `json:"last_classified"`
	ClassificationCount int64         `json:"classification_count"` // Number of times classified
	Network             string        `json:"network"`

	// Activity Metrics
	TotalTransactions    int64     `json:"total_transactions"`
	TotalVolume          string    `json:"total_volume"`
	UniqueCounterparties int64     `json:"unique_counterparties"`
	FirstActivity        time.Time `json:"first_activity"`
	LastActivity         time.Time `json:"last_activity"`

	// Risk Indicators
	SuspiciousActivities []string `json:"suspicious_activities"`
	BlacklistReasons     []string `json:"blacklist_reasons"`
	SanctionDetails      []string `json:"sanction_details"`
	ReportedBy           []string `json:"reported_by"` // Who reported this entity

	// Verification
	IsVerified         bool      `json:"is_verified"`
	VerificationSource string    `json:"verification_source"`
	VerifiedBy         string    `json:"verified_by"`
	VerificationDate   time.Time `json:"verification_date"`
}

// NodeRelationship represents relationships between different nodes
type NodeRelationship struct {
	FromAddress      string    `json:"from_address"`
	ToAddress        string    `json:"to_address"`
	RelationshipType string    `json:"relationship_type"` // "FUNDING", "CONTROLLED_BY", "SIMILAR_PATTERN", etc.
	Strength         float64   `json:"strength"`          // 0.0 - 1.0
	TotalValue       string    `json:"total_value"`
	TransactionCount int64     `json:"transaction_count"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	Network          string    `json:"network"`

	// Relationship metadata
	Confidence      float64                `json:"confidence"`
	DetectionMethod string                 `json:"detection_method"`
	Properties      map[string]interface{} `json:"properties"`
}

// GetNodeTypeCategory returns the broad category of a node type
func (nt NodeType) GetNodeTypeCategory() string {
	switch nt {
	case NodeTypeEOA, NodeTypeExchangeWallet, NodeTypeExchangeHotWallet, NodeTypeExchangeColdWallet,
		NodeTypeBridgeWallet, NodeTypeMEVBot, NodeTypeArbitrageBot, NodeTypeMarketMaker,
		NodeTypeWhale, NodeTypeCEXDeposit, NodeTypeCEXWithdrawal, NodeTypeCEXSettlement:
		return "WALLET"
	case NodeTypeDEXContract, NodeTypeLendingContract, NodeTypeStakingContract, NodeTypeNFTMarketplace,
		NodeTypeDAOContract, NodeTypeTokenContract, NodeTypeBridgeContract, NodeTypeYieldContract,
		NodeTypeInsuranceContract, NodeTypeOracleContract, NodeTypeMultisigContract, NodeTypeProxyContract,
		NodeTypeFactoryContract:
		return "CONTRACT"
	case NodeTypeMixerWallet, NodeTypeSuspiciousWallet, NodeTypeBlacklistedWallet, NodeTypeGamblingContract,
		NodeTypePonziContract, NodeTypeDarkWeb, NodeTypeRansomware, NodeTypeTerroristFinancing,
		NodeTypeMoneyLaundering, NodeTypeSanctioned:
		return "HIGH_RISK"
	case NodeTypeOracleService, NodeTypeFlashLoanProvider, NodeTypeYieldAggregator,
		NodeTypeLiquidityProvider, NodeTypeValidator, NodeTypeMiner:
		return "SERVICE"
	case NodeTypePrivacyContract:
		return "PRIVACY"
	default:
		return "OTHER"
	}
}

// GetDefaultRiskLevel returns the default risk level for a node type
func (nt NodeType) GetDefaultRiskLevel() NodeRiskLevel {
	switch nt {
	case NodeTypeBlacklistedWallet, NodeTypePonziContract, NodeTypeRansomware,
		NodeTypeTerroristFinancing, NodeTypeSanctioned:
		return RiskLevelCritical
	case NodeTypeMixerWallet, NodeTypeSuspiciousWallet, NodeTypeGamblingContract,
		NodeTypeDarkWeb, NodeTypeMoneyLaundering:
		return RiskLevelHigh
	case NodeTypeMEVBot, NodeTypeArbitrageBot, NodeTypePrivacyContract:
		return RiskLevelMedium
	case NodeTypeEOA, NodeTypeExchangeWallet, NodeTypeDEXContract, NodeTypeLendingContract,
		NodeTypeStakingContract, NodeTypeTokenContract:
		return RiskLevelLow
	default:
		return RiskLevelUnknown
	}
}

// IsExchangeRelated checks if the node type is related to centralized exchanges
func (nt NodeType) IsExchangeRelated() bool {
	return nt == NodeTypeExchangeWallet || nt == NodeTypeExchangeHotWallet ||
		nt == NodeTypeExchangeColdWallet || nt == NodeTypeCEXDeposit ||
		nt == NodeTypeCEXWithdrawal || nt == NodeTypeCEXSettlement
}

// IsContractType checks if the node type represents a smart contract
func (nt NodeType) IsContractType() bool {
	return nt.GetNodeTypeCategory() == "CONTRACT" || nt == NodeTypePrivacyContract ||
		nt == NodeTypeGamblingContract || nt == NodeTypePonziContract
}

// IsHighRisk checks if the node type is considered high risk
func (nt NodeType) IsHighRisk() bool {
	level := nt.GetDefaultRiskLevel()
	return level == RiskLevelHigh || level == RiskLevelCritical
}

// GetTypicalBehaviorPatterns returns typical behavior patterns for this node type
func (nt NodeType) GetTypicalBehaviorPatterns() []string {
	switch nt {
	case NodeTypeMEVBot:
		return []string{"high_frequency_trading", "flashloan_usage", "arbitrage_patterns"}
	case NodeTypeArbitrageBot:
		return []string{"cross_exchange_trading", "price_difference_exploitation"}
	case NodeTypeExchangeHotWallet:
		return []string{"high_volume_transactions", "batch_processing", "regular_consolidation"}
	case NodeTypeMixerWallet:
		return []string{"obfuscation_patterns", "multi_hop_transactions", "timing_analysis_resistance"}
	case NodeTypeWhale:
		return []string{"large_value_transactions", "market_impact", "holding_patterns"}
	case NodeTypeDEXContract:
		return []string{"swap_operations", "liquidity_provision", "fee_collection"}
	case NodeTypeLendingContract:
		return []string{"deposit_withdraw_cycles", "interest_accrual", "liquidations"}
	default:
		return []string{}
	}
}

// NodeClassificationRule represents rules for node classification
type NodeClassificationRule struct {
	NodeType         NodeType `json:"node_type"`
	RequiredPatterns []string `json:"required_patterns"` // Must match all
	OptionalPatterns []string `json:"optional_patterns"` // Nice to have
	ExcludePatterns  []string `json:"exclude_patterns"`  // Must not match
	MinTransactions  int64    `json:"min_transactions"`
	MinVolume        string   `json:"min_volume"`
	MinConfidence    float64  `json:"min_confidence"`
	Weight           float64  `json:"weight"`
	TimeframeHours   int      `json:"timeframe_hours"` // Analysis timeframe
}

// DetectionMethod represents different methods used for node classification
type DetectionMethod string

const (
	DetectionMethodPatternAnalysis  DetectionMethod = "PATTERN_ANALYSIS"
	DetectionMethodMLModel          DetectionMethod = "ML_MODEL"
	DetectionMethodManual           DetectionMethod = "MANUAL"
	DetectionMethodOSINT            DetectionMethod = "OSINT" // Open Source Intelligence
	DetectionMethodBlocklist        DetectionMethod = "BLOCKLIST"
	DetectionMethodHeuristic        DetectionMethod = "HEURISTIC"
	DetectionMethodCommunity        DetectionMethod = "COMMUNITY"
	DetectionMethodExchange         DetectionMethod = "EXCHANGE_LABEL"
	DetectionMethodBytecodeAnalysis DetectionMethod = "BYTECODE_ANALYSIS" // New method for contract detection
)

// RelationshipType represents different types of relationships between nodes
type RelationshipType string

const (
	RelationshipFunding        RelationshipType = "FUNDING"
	RelationshipControlledBy   RelationshipType = "CONTROLLED_BY"
	RelationshipSimilarPattern RelationshipType = "SIMILAR_PATTERN"
	RelationshipCluster        RelationshipType = "CLUSTER"
	RelationshipPartnership    RelationshipType = "PARTNERSHIP"
	RelationshipSuspicious     RelationshipType = "SUSPICIOUS"
	RelationshipBlacklisted    RelationshipType = "BLACKLISTED"
)
