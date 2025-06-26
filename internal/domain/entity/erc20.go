package entity

import (
	"time"
)

// ERC20Transfer represents an ERC20 Transfer event
type ERC20Transfer struct {
	ContractAddress string    `json:"contract_address"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Value           string    `json:"value"`
	TxHash          string    `json:"tx_hash"`
	BlockNumber     string    `json:"block_number"`
	Timestamp       time.Time `json:"timestamp"`
	Network         string    `json:"network"`
}

// ERC20Contract represents an ERC20 contract
type ERC20Contract struct {
	Address   string    `json:"address"`
	Name      string    `json:"name"`
	Symbol    string    `json:"symbol"`
	Decimals  int       `json:"decimals"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	TotalTxs  int64     `json:"total_txs"`
	Network   string    `json:"network"`
}

// ERC20TransferRelationship represents a transfer relationship between two wallets via ERC20 token
type ERC20TransferRelationship struct {
	FromAddress     string    `json:"from_address"`
	ToAddress       string    `json:"to_address"`
	ContractAddress string    `json:"contract_address"`
	Value           string    `json:"value"`
	TxHash          string    `json:"tx_hash"`
	Timestamp       time.Time `json:"timestamp"`
	Network         string    `json:"network"`
}
