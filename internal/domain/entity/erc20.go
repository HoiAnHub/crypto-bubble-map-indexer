package entity

import (
	"time"
)

// ContractInteractionType represents the type of contract interaction
type ContractInteractionType string

const (
	// ERC20 Standard Operations
	InteractionTransfer          ContractInteractionType = "TRANSFER"
	InteractionTransferFrom      ContractInteractionType = "TRANSFER_FROM"
	InteractionApprove           ContractInteractionType = "APPROVE"
	InteractionIncreaseAllowance ContractInteractionType = "INCREASE_ALLOWANCE"
	InteractionDecreaseAllowance ContractInteractionType = "DECREASE_ALLOWANCE"

	// DeFi Operations
	InteractionSwap            ContractInteractionType = "SWAP"
	InteractionAddLiquidity    ContractInteractionType = "ADD_LIQUIDITY"
	InteractionRemoveLiquidity ContractInteractionType = "REMOVE_LIQUIDITY"
	InteractionDeposit         ContractInteractionType = "DEPOSIT"
	InteractionWithdraw        ContractInteractionType = "WITHDRAW"
	InteractionMulticall       ContractInteractionType = "MULTICALL"

	// Special Cases
	InteractionETHTransfer     ContractInteractionType = "ETH_TRANSFER"
	InteractionUnknownContract ContractInteractionType = "UNKNOWN_CONTRACT_CALL"
)

// ERC20Transfer represents an ERC20 Transfer or contract interaction event
type ERC20Transfer struct {
	ContractAddress string                  `json:"contract_address"`
	From            string                  `json:"from"`
	To              string                  `json:"to"`
	Value           string                  `json:"value"`
	TxHash          string                  `json:"tx_hash"`
	BlockNumber     string                  `json:"block_number"`
	Timestamp       time.Time               `json:"timestamp"`
	Network         string                  `json:"network"`
	InteractionType ContractInteractionType `json:"interaction_type"` // New field for interaction type
	MethodSignature string                  `json:"method_signature"` // New field for method signature
	Success         bool                    `json:"success"`          // New field for transaction success
}

// ERC20Contract represents an ERC20 contract
type ERC20Contract struct {
	Address      string    `json:"address"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	Decimals     int       `json:"decimals"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	TotalTxs     int64     `json:"total_txs"`
	Network      string    `json:"network"`
	ContractType string    `json:"contract_type"` // New field: "ERC20", "DEX", "LENDING", etc.
	IsVerified   bool      `json:"is_verified"`   // New field for contract verification status
}

// ERC20TransferRelationship represents a transfer relationship between two wallets via ERC20 token
type ERC20TransferRelationship struct {
	FromAddress      string                  `json:"from_address"`
	ToAddress        string                  `json:"to_address"`
	ContractAddress  string                  `json:"contract_address"`
	Value            string                  `json:"value"`
	TxHash           string                  `json:"tx_hash"`
	Timestamp        time.Time               `json:"timestamp"`
	Network          string                  `json:"network"`
	InteractionType  ContractInteractionType `json:"interaction_type"`  // New field
	TotalValue       string                  `json:"total_value"`       // New field for aggregated value
	TransactionCount int64                   `json:"transaction_count"` // New field for count
	FirstInteraction time.Time               `json:"first_interaction"` // New field
	LastInteraction  time.Time               `json:"last_interaction"`  // New field
}

// ContractInteractionRelationship represents generic contract interactions
type ContractInteractionRelationship struct {
	FromAddress      string                  `json:"from_address"`
	ContractAddress  string                  `json:"contract_address"`
	InteractionType  ContractInteractionType `json:"interaction_type"`
	TotalValue       string                  `json:"total_value"`
	TransactionCount int64                   `json:"transaction_count"`
	FirstInteraction time.Time               `json:"first_interaction"`
	LastInteraction  time.Time               `json:"last_interaction"`
	Network          string                  `json:"network"`
}

// GetRelationshipType returns the Neo4J relationship type based on interaction type
func (c ContractInteractionType) GetRelationshipType() string {
	switch c {
	case InteractionTransfer, InteractionTransferFrom:
		return "ERC20_TRANSFER"
	case InteractionApprove, InteractionIncreaseAllowance, InteractionDecreaseAllowance:
		return "ERC20_APPROVAL"
	case InteractionSwap:
		return "DEX_SWAP"
	case InteractionAddLiquidity, InteractionRemoveLiquidity:
		return "LIQUIDITY_OPERATION"
	case InteractionDeposit, InteractionWithdraw:
		return "DEFI_OPERATION"
	case InteractionMulticall:
		return "MULTICALL_OPERATION"
	case InteractionETHTransfer:
		return "ETH_TRANSFER"
	case InteractionUnknownContract:
		return "CONTRACT_INTERACTION"
	default:
		return "CONTRACT_INTERACTION"
	}
}

// ContractType represents different types of smart contracts
type ContractType string

const (
	// Standard Contract Types
	ContractTypeERC20   ContractType = "ERC20"
	ContractTypeERC721  ContractType = "ERC721"
	ContractTypeERC1155 ContractType = "ERC1155"

	// DeFi Protocol Types
	ContractTypeDEX         ContractType = "DEX"
	ContractTypeAMM         ContractType = "AMM" // Automated Market Maker
	ContractTypeLendingPool ContractType = "LENDING_POOL"
	ContractTypeYieldFarm   ContractType = "YIELD_FARM"
	ContractTypeVault       ContractType = "VAULT"
	ContractTypeStaking     ContractType = "STAKING"

	// DEX Specific Types
	ContractTypeUniswapV2   ContractType = "UNISWAP_V2"
	ContractTypeUniswapV3   ContractType = "UNISWAP_V3"
	ContractTypeSushiSwap   ContractType = "SUSHISWAP"
	ContractTypePancakeSwap ContractType = "PANCAKESWAP"
	ContractType1inch       ContractType = "1INCH_AGGREGATOR"

	// Lending Protocols
	ContractTypeCompound ContractType = "COMPOUND"
	ContractTypeAave     ContractType = "AAVE"
	ContractTypeMakerDAO ContractType = "MAKERDAO"

	// Bridge & Layer 2
	ContractTypeBridge    ContractType = "BRIDGE"
	ContractTypeL2Gateway ContractType = "L2_GATEWAY"

	// Utility Contracts
	ContractTypeMulticall ContractType = "MULTICALL"
	ContractTypeProxy     ContractType = "PROXY"
	ContractTypeWETH      ContractType = "WETH"

	// Unknown/Generic
	ContractTypeUnknown ContractType = "UNKNOWN"
	ContractTypeGeneric ContractType = "GENERIC_CONTRACT"
)

// ContractClassification represents comprehensive contract classification data
type ContractClassification struct {
	Address             string                          `json:"address"`
	PrimaryType         ContractType                    `json:"primary_type"`
	SecondaryTypes      []ContractType                  `json:"secondary_types"`
	ConfidenceScore     float64                         `json:"confidence_score"`     // 0.0 - 1.0
	DetectedProtocols   []string                        `json:"detected_protocols"`   // ["uniswap", "compound"]
	MethodSignatures    map[string]int                  `json:"method_signatures"`    // signature -> count
	InteractionPatterns map[ContractInteractionType]int `json:"interaction_patterns"` // pattern -> count
	TotalInteractions   int64                           `json:"total_interactions"`
	UniqueUsers         int64                           `json:"unique_users"`
	FirstSeen           time.Time                       `json:"first_seen"`
	LastSeen            time.Time                       `json:"last_seen"`
	IsVerified          bool                            `json:"is_verified"`
	VerificationSource  string                          `json:"verification_source"` // "etherscan", "manual", "heuristic"
	Tags                []string                        `json:"tags"`                // ["dex", "high-volume", "popular"]
	Network             string                          `json:"network"`
}

// ClassificationRule represents rules for contract classification
type ClassificationRule struct {
	ContractType        ContractType              `json:"contract_type"`
	RequiredMethods     []string                  `json:"required_methods"` // Must have all
	OptionalMethods     []string                  `json:"optional_methods"` // Nice to have
	ExcludeMethods      []string                  `json:"exclude_methods"`  // Must not have
	InteractionPatterns []ContractInteractionType `json:"interaction_patterns"`
	MinConfidence       float64                   `json:"min_confidence"`
	Weight              float64                   `json:"weight"` // Rule importance
}

// GetContractTypeCategory returns the broad category of a contract type
func (ct ContractType) GetContractTypeCategory() string {
	switch ct {
	case ContractTypeDEX, ContractTypeAMM, ContractTypeUniswapV2, ContractTypeUniswapV3,
		ContractTypeSushiSwap, ContractTypePancakeSwap, ContractType1inch:
		return "DEX"
	case ContractTypeLendingPool, ContractTypeCompound, ContractTypeAave, ContractTypeMakerDAO:
		return "LENDING"
	case ContractTypeYieldFarm, ContractTypeVault, ContractTypeStaking:
		return "YIELD"
	case ContractTypeERC20, ContractTypeERC721, ContractTypeERC1155:
		return "TOKEN"
	case ContractTypeBridge, ContractTypeL2Gateway:
		return "BRIDGE"
	case ContractTypeMulticall, ContractTypeProxy, ContractTypeWETH:
		return "UTILITY"
	default:
		return "OTHER"
	}
}

// IsDefiProtocol checks if the contract type is a DeFi protocol
func (ct ContractType) IsDefiProtocol() bool {
	category := ct.GetContractTypeCategory()
	return category == "DEX" || category == "LENDING" || category == "YIELD"
}

// GetTypicalMethods returns typical method signatures for this contract type
func (ct ContractType) GetTypicalMethods() []string {
	switch ct {
	case ContractTypeDEX, ContractTypeAMM:
		return []string{"7ff36ab5", "18cbafe5", "38ed1739", "e8e33700", "baa2abde"} // swap methods
	case ContractTypeUniswapV2:
		return []string{"7ff36ab5", "18cbafe5", "38ed1739", "e8e33700", "baa2abde", "022c0d9f"} // + swap exact
	case ContractTypeLendingPool, ContractTypeCompound:
		return []string{"d0e30db0", "2e1a7d4d", "a6afed95", "852a12e3"} // deposit, withdraw, mint, redeem
	case ContractTypeAave:
		return []string{"d65d7f80", "69328dec", "e8eda9df"} // deposit, withdraw, borrow
	case ContractTypeWETH:
		return []string{"d0e30db0", "2e1a7d4d"} // deposit, withdraw
	case ContractTypeMulticall:
		return []string{"ac9650d8", "5ae401dc"} // multicall, multicallWithDeadline
	default:
		return []string{}
	}
}
