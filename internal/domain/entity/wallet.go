package entity

import (
	"time"
)

// Wallet represents an Ethereum wallet/address in Neo4J
type Wallet struct {
	Address           string    `json:"address"`
	FirstSeen         time.Time `json:"first_seen"`
	LastSeen          time.Time `json:"last_seen"`
	TotalTransactions int64     `json:"total_transactions"`
	TotalSent         string    `json:"total_sent"`
	TotalReceived     string    `json:"total_received"`
	Network           string    `json:"network"`
}

// WalletStats represents statistics for a wallet
type WalletStats struct {
	Address             string `json:"address"`
	IncomingConnections int64  `json:"incoming_connections"`
	OutgoingConnections int64  `json:"outgoing_connections"`
	TotalVolume         string `json:"total_volume"`
	TransactionCount    int64  `json:"transaction_count"`
}

// WalletConnection represents a connection between two wallets
type WalletConnection struct {
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	TotalValue  string    `json:"total_value"`
	TxCount     int64     `json:"tx_count"`
	FirstTx     time.Time `json:"first_tx"`
	LastTx      time.Time `json:"last_tx"`
}
