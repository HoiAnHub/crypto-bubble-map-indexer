package repository

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
)

// NodeClassificationRepository defines methods for managing node classifications
type NodeClassificationRepository interface {
	// CreateOrUpdateClassification creates or updates a node classification
	CreateOrUpdateClassification(ctx context.Context, classification *entity.NodeClassification) error

	// GetClassification retrieves a node classification by address
	GetClassification(ctx context.Context, address string) (*entity.NodeClassification, error)

	// GetClassificationsByType retrieves all nodes of a specific type
	GetClassificationsByType(ctx context.Context, nodeType entity.NodeType) ([]*entity.NodeClassification, error)

	// GetClassificationsByRiskLevel retrieves all nodes with a specific risk level
	GetClassificationsByRiskLevel(ctx context.Context, riskLevel entity.NodeRiskLevel) ([]*entity.NodeClassification, error)

	// SearchClassifications searches for classifications based on criteria
	SearchClassifications(ctx context.Context, criteria *ClassificationSearchCriteria) ([]*entity.NodeClassification, error)

	// UpdateRiskLevel updates the risk level for a specific address
	UpdateRiskLevel(ctx context.Context, address string, riskLevel entity.NodeRiskLevel, reason string) error

	// AddToBlacklist adds an address to the blacklist
	AddToBlacklist(ctx context.Context, address, reason string) error

	// RemoveFromBlacklist removes an address from the blacklist
	RemoveFromBlacklist(ctx context.Context, address string) error

	// GetBlacklistedAddresses retrieves all blacklisted addresses
	GetBlacklistedAddresses(ctx context.Context) (map[string]string, error)

	// CreateNodeRelationship creates a relationship between two nodes
	CreateNodeRelationship(ctx context.Context, relationship *entity.NodeRelationship) error

	// GetNodeRelationships retrieves relationships for a specific address
	GetNodeRelationships(ctx context.Context, address string) ([]*entity.NodeRelationship, error)

	// GetSuspiciousCluster identifies clusters of suspicious nodes
	GetSuspiciousCluster(ctx context.Context, address string, maxDepth int) ([]*entity.NodeClassification, error)

	// GetExchangeWallets retrieves all wallets associated with a specific exchange
	GetExchangeWallets(ctx context.Context, exchange string) ([]*entity.NodeClassification, error)

	// GetHighRiskNodes retrieves all high-risk and critical nodes
	GetHighRiskNodes(ctx context.Context) ([]*entity.NodeClassification, error)

	// UpdateClassificationStats updates classification statistics
	UpdateClassificationStats(ctx context.Context, address string, stats *entity.WalletStats) error

	// GetClassificationHistory retrieves the classification history for an address
	GetClassificationHistory(ctx context.Context, address string) ([]*ClassificationHistoryEntry, error)

	// BulkUpdateClassifications updates multiple classifications in a batch
	BulkUpdateClassifications(ctx context.Context, classifications []*entity.NodeClassification) error
}

// ClassificationSearchCriteria defines search criteria for node classifications
type ClassificationSearchCriteria struct {
	NodeTypes             []entity.NodeType      `json:"node_types"`
	RiskLevels            []entity.NodeRiskLevel `json:"risk_levels"`
	Tags                  []string               `json:"tags"`
	Exchanges             []string               `json:"exchanges"`
	Protocols             []string               `json:"protocols"`
	MinConfidenceScore    float64                `json:"min_confidence_score"`
	MinTransactions       int64                  `json:"min_transactions"`
	MinVolume             string                 `json:"min_volume"`
	HasSuspiciousActivity bool                   `json:"has_suspicious_activity"`
	IsBlacklisted         bool                   `json:"is_blacklisted"`
	IsSanctioned          bool                   `json:"is_sanctioned"`
	Limit                 int                    `json:"limit"`
	Offset                int                    `json:"offset"`
}

// ClassificationHistoryEntry represents a historical classification entry
type ClassificationHistoryEntry struct {
	Address           string               `json:"address"`
	PreviousType      entity.NodeType      `json:"previous_type"`
	NewType           entity.NodeType      `json:"new_type"`
	PreviousRiskLevel entity.NodeRiskLevel `json:"previous_risk_level"`
	NewRiskLevel      entity.NodeRiskLevel `json:"new_risk_level"`
	ConfidenceScore   float64              `json:"confidence_score"`
	DetectionMethod   string               `json:"detection_method"`
	Reason            string               `json:"reason"`
	Timestamp         string               `json:"timestamp"`
	ClassifiedBy      string               `json:"classified_by"`
}
