package entity

import (
	"time"
)

// Transaction represents an Ethereum transaction event from NATS
type Transaction struct {
	Hash        string    `json:"hash"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Value       string    `json:"value"`
	Data        string    `json:"data"`
	BlockNumber string    `json:"block_number"`
	BlockHash   string    `json:"block_hash"`
	Timestamp   time.Time `json:"timestamp"`
	GasUsed     string    `json:"gas_used"`
	GasPrice    string    `json:"gas_price"`
	Network     string    `json:"network"`
}

// TransactionNode represents a transaction node in Neo4J
type TransactionNode struct {
	Hash        string    `json:"hash"`
	BlockNumber string    `json:"block_number"`
	Value       string    `json:"value"`
	GasUsed     string    `json:"gas_used"`
	GasPrice    string    `json:"gas_price"`
	Timestamp   time.Time `json:"timestamp"`
	Network     string    `json:"network"`
}

// TransactionRelationship represents a relationship between wallets via transaction
type TransactionRelationship struct {
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	Value       string    `json:"value"`
	GasPrice    string    `json:"gas_price"`
	Timestamp   time.Time `json:"timestamp"`
	TxHash      string    `json:"tx_hash"`
}
