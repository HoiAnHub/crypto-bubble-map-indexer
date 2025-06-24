package service

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
)

// IndexingService defines the interface for indexing operations
type IndexingService interface {
	// ProcessTransaction processes a transaction event and indexes it
	ProcessTransaction(ctx context.Context, tx *entity.Transaction) error

	// ProcessTransactionBatch processes multiple transactions in batch
	ProcessTransactionBatch(ctx context.Context, transactions []*entity.Transaction) error

	// GetWalletAnalytics retrieves analytics for a wallet
	GetWalletAnalytics(ctx context.Context, address string) (*entity.WalletStats, error)

	// GetBubbleAnalysis retrieves bubble analysis data
	GetBubbleAnalysis(ctx context.Context, minConnections int, limit int) ([]*entity.Wallet, error)

	// GetTransactionPath finds transaction path between wallets
	GetTransactionPath(ctx context.Context, fromAddress, toAddress string, maxHops int) ([]*entity.TransactionNode, error)
}
