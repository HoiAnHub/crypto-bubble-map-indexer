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

	// GetERC20TransfersForWallet retrieves ERC20 transfers for a wallet
	GetERC20TransfersForWallet(ctx context.Context, address string, limit int) ([]*entity.ERC20Transfer, error)

	// GetERC20TransfersBetweenWallets retrieves ERC20 transfers between two wallets
	GetERC20TransfersBetweenWallets(ctx context.Context, fromAddress, toAddress string, limit int) ([]*entity.ERC20Transfer, error)
}
