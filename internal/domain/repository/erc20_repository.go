package repository

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
)

// ERC20Repository defines the interface for ERC20 related database operations
type ERC20Repository interface {
	// CreateOrUpdateERC20Contract creates or updates an ERC20 contract
	CreateOrUpdateERC20Contract(ctx context.Context, contract *entity.ERC20Contract) error

	// CreateERC20TransferRelationship creates a transfer relationship between wallets
	CreateERC20TransferRelationship(ctx context.Context, transfer *entity.ERC20TransferRelationship) error

	// BatchCreateERC20TransferRelationships creates multiple transfer relationships in batch
	BatchCreateERC20TransferRelationships(ctx context.Context, transfers []*entity.ERC20TransferRelationship) error

	// GetERC20Contract retrieves an ERC20 contract by address
	GetERC20Contract(ctx context.Context, address string) (*entity.ERC20Contract, error)

	// GetERC20TransfersBetweenWallets retrieves ERC20 transfers between two wallets
	GetERC20TransfersBetweenWallets(ctx context.Context, fromAddress, toAddress string, limit int) ([]*entity.ERC20Transfer, error)

	// GetERC20TransfersForWallet retrieves all ERC20 transfers for a wallet
	GetERC20TransfersForWallet(ctx context.Context, address string, limit int) ([]*entity.ERC20Transfer, error)

	// Contract Classification Methods
	// StoreContractClassification stores contract classification data
	StoreContractClassification(ctx context.Context, classification *entity.ContractClassification) error

	// GetContractClassification retrieves contract classification data
	GetContractClassification(ctx context.Context, contractAddress string) (*entity.ContractClassification, error)

	// UpdateContractClassification updates existing contract classification
	UpdateContractClassification(ctx context.Context, classification *entity.ContractClassification) error

	// GetContractsByType retrieves contracts by type
	GetContractsByType(ctx context.Context, contractType entity.ContractType, limit int) ([]*entity.ERC20Contract, error)

	// GetContractClassificationStats retrieves classification statistics
	GetContractClassificationStats(ctx context.Context) (map[entity.ContractType]int, error)
}
