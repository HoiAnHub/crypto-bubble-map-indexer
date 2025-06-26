package service

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
)

// ERC20DecoderService defines the interface for ERC20 decoding operations
type ERC20DecoderService interface {
	// DecodeERC20Transfer decodes ERC20 Transfer events from transaction data
	DecodeERC20Transfer(ctx context.Context, tx *entity.Transaction) ([]*entity.ERC20Transfer, error)

	// IsERC20Contract checks if an address is an ERC20 contract
	IsERC20Contract(ctx context.Context, address string) (bool, error)

	// GetERC20ContractInfo retrieves ERC20 contract information
	GetERC20ContractInfo(ctx context.Context, address string) (*entity.ERC20Contract, error)
}
