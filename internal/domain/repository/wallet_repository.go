package repository

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
)

// WalletRepository defines the interface for wallet data operations
type WalletRepository interface {
	// CreateOrUpdateWallet creates a new wallet or updates existing one
	CreateOrUpdateWallet(ctx context.Context, wallet *entity.Wallet) error

	// GetWallet retrieves a wallet by address
	GetWallet(ctx context.Context, address string) (*entity.Wallet, error)

	// GetWalletStats retrieves statistics for a wallet
	GetWalletStats(ctx context.Context, address string) (*entity.WalletStats, error)

	// GetWalletConnections retrieves connections for a wallet
	GetWalletConnections(ctx context.Context, address string, limit int) ([]*entity.WalletConnection, error)

	// FindConnectedWallets finds wallets connected to a given wallet within specified hops
	FindConnectedWallets(ctx context.Context, address string, maxHops int) ([]*entity.Wallet, error)

	// GetTopWallets retrieves top wallets by transaction count
	GetTopWallets(ctx context.Context, limit int) ([]*entity.Wallet, error)

	// GetBubbleWallets retrieves wallets that form bubbles (high connectivity)
	GetBubbleWallets(ctx context.Context, minConnections int, limit int) ([]*entity.Wallet, error)
}
