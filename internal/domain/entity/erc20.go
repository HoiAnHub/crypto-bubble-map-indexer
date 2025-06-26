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
