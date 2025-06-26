package service

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
)

// ContractClassifierService defines the interface for contract classification
type ContractClassifierService interface {
	// ClassifyContract analyzes contract interactions to determine contract type
	ClassifyContract(ctx context.Context, contractAddress string, interactions []*entity.ERC20Transfer) (*entity.ContractClassification, error)

	// ClassifyFromMethodSignature provides quick classification based on method signature
	ClassifyFromMethodSignature(methodSignature string) entity.ContractType

	// UpdateContractClassification updates contract classification based on new interactions
	UpdateContractClassification(ctx context.Context, classification *entity.ContractClassification, newInteraction *entity.ERC20Transfer) *entity.ContractClassification

	// GetContractPatterns returns known patterns for contract types
	GetContractPatterns() map[entity.ContractType][]string
}
