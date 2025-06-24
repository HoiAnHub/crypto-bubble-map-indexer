package repository

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
)

// TransactionRepository defines the interface for transaction data operations
type TransactionRepository interface {
	// CreateTransaction creates a new transaction node
	CreateTransaction(ctx context.Context, tx *entity.TransactionNode) error

	// CreateTransactionRelationship creates a relationship between wallets via transaction
	CreateTransactionRelationship(ctx context.Context, rel *entity.TransactionRelationship) error

	// GetTransaction retrieves a transaction by hash
	GetTransaction(ctx context.Context, hash string) (*entity.TransactionNode, error)

	// GetTransactionPath finds the path between two wallets through transactions
	GetTransactionPath(ctx context.Context, fromAddress, toAddress string, maxHops int) ([]*entity.TransactionNode, error)

	// GetTransactionsByWallet retrieves transactions for a specific wallet
	GetTransactionsByWallet(ctx context.Context, address string, limit int) ([]*entity.TransactionNode, error)

	// GetTransactionsByTimeRange retrieves transactions within a time range
	GetTransactionsByTimeRange(ctx context.Context, startTime, endTime string, limit int) ([]*entity.TransactionNode, error)

	// BatchCreateTransactions creates multiple transactions in a batch
	BatchCreateTransactions(ctx context.Context, transactions []*entity.TransactionNode) error

	// BatchCreateRelationships creates multiple relationships in a batch
	BatchCreateRelationships(ctx context.Context, relationships []*entity.TransactionRelationship) error
}
