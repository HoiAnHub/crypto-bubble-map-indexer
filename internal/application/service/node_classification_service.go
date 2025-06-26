package service

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/domain/repository"
	domainService "crypto-bubble-map-indexer/internal/domain/service"
	"fmt"
	"log"
	"time"
)

// NodeClassificationAppService handles node classification at application level
type NodeClassificationAppService struct {
	nodeClassifier     *domainService.NodeClassifierService
	walletRepo         repository.WalletRepository
	classificationRepo repository.NodeClassificationRepository
	transactionRepo    repository.TransactionRepository
}

// NewNodeClassificationAppService creates a new node classification application service
func NewNodeClassificationAppService(
	nodeClassifier *domainService.NodeClassifierService,
	walletRepo repository.WalletRepository,
	classificationRepo repository.NodeClassificationRepository,
	transactionRepo repository.TransactionRepository,
) *NodeClassificationAppService {
	return &NodeClassificationAppService{
		nodeClassifier:     nodeClassifier,
		walletRepo:         walletRepo,
		classificationRepo: classificationRepo,
		transactionRepo:    transactionRepo,
	}
}

// ClassifyWalletAddress classifies a wallet address and saves the result
func (s *NodeClassificationAppService) ClassifyWalletAddress(ctx context.Context, address string) (*entity.NodeClassification, error) {
	log.Printf("Classifying address: %s", address)

	// Get wallet statistics
	stats, err := s.walletRepo.GetWalletStats(ctx, address)
	if err != nil {
		log.Printf("Error getting wallet stats for %s: %v", address, err)
		// Continue with classification even if stats unavailable
		stats = nil
	}

	// Get behavioral patterns (simplified - in real implementation, this would analyze transaction patterns)
	patterns := s.generateBehavioralPatterns(ctx, address, stats)

	// Classify the address
	classification, err := s.nodeClassifier.ClassifyNode(ctx, address, stats, patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to classify node %s: %w", address, err)
	}

	// Update the wallet with classification info
	if err := s.updateWalletWithClassification(ctx, address, classification); err != nil {
		log.Printf("Warning: failed to update wallet classification for %s: %v", address, err)
	}

	// Save classification to repository
	if err := s.classificationRepo.CreateOrUpdateClassification(ctx, classification); err != nil {
		return nil, fmt.Errorf("failed to save classification for %s: %w", address, err)
	}

	log.Printf("Successfully classified %s as %s (risk: %s, confidence: %.2f)",
		address, classification.PrimaryType, classification.RiskLevel, classification.ConfidenceScore)

	return classification, nil
}

// BulkClassifyAddresses classifies multiple addresses in batch
func (s *NodeClassificationAppService) BulkClassifyAddresses(ctx context.Context, addresses []string) ([]*entity.NodeClassification, error) {
	log.Printf("Starting bulk classification of %d addresses", len(addresses))

	classifications := make([]*entity.NodeClassification, 0, len(addresses))

	for i, address := range addresses {
		if i > 0 && i%100 == 0 {
			log.Printf("Processed %d/%d addresses", i, len(addresses))
		}

		classification, err := s.ClassifyWalletAddress(ctx, address)
		if err != nil {
			log.Printf("Failed to classify address %s: %v", address, err)
			continue
		}

		classifications = append(classifications, classification)
	}

	log.Printf("Completed bulk classification: %d/%d addresses successfully classified",
		len(classifications), len(addresses))

	return classifications, nil
}

// ReClassifyAddress re-classifies an address with updated data
func (s *NodeClassificationAppService) ReClassifyAddress(ctx context.Context, address string) (*entity.NodeClassification, error) {
	log.Printf("Re-classifying address: %s", address)

	// Get existing classification
	existing, err := s.classificationRepo.GetClassification(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing classification: %w", err)
	}

	// Perform new classification
	newClassification, err := s.ClassifyWalletAddress(ctx, address)
	if err != nil {
		return nil, err
	}

	// Compare with existing classification
	if existing != nil {
		s.logClassificationChanges(address, existing, newClassification)
	}

	return newClassification, nil
}

// GetSuspiciousAddresses returns addresses that match suspicious criteria
func (s *NodeClassificationAppService) GetSuspiciousAddresses(ctx context.Context) ([]*entity.NodeClassification, error) {
	criteria := &repository.ClassificationSearchCriteria{
		RiskLevels: []entity.NodeRiskLevel{
			entity.RiskLevelHigh,
			entity.RiskLevelCritical,
		},
		HasSuspiciousActivity: true,
		Limit:                 1000,
	}

	return s.classificationRepo.SearchClassifications(ctx, criteria)
}

// GetExchangeWallets returns all wallets associated with exchanges
func (s *NodeClassificationAppService) GetExchangeWallets(ctx context.Context, exchange string) ([]*entity.NodeClassification, error) {
	if exchange == "" {
		// Get all exchange-related wallets
		criteria := &repository.ClassificationSearchCriteria{
			NodeTypes: []entity.NodeType{
				entity.NodeTypeExchangeWallet,
				entity.NodeTypeExchangeHotWallet,
				entity.NodeTypeExchangeColdWallet,
				entity.NodeTypeCEXDeposit,
				entity.NodeTypeCEXWithdrawal,
				entity.NodeTypeCEXSettlement,
			},
			Limit: 1000,
		}
		return s.classificationRepo.SearchClassifications(ctx, criteria)
	}

	return s.classificationRepo.GetExchangeWallets(ctx, exchange)
}

// GetHighValueTransactionNodes returns nodes involved in high-value transactions
func (s *NodeClassificationAppService) GetHighValueTransactionNodes(ctx context.Context, minValue string) ([]*entity.NodeClassification, error) {
	criteria := &repository.ClassificationSearchCriteria{
		MinVolume: minValue,
		NodeTypes: []entity.NodeType{
			entity.NodeTypeWhale,
			entity.NodeTypeExchangeHotWallet,
			entity.NodeTypeDEXContract,
		},
		Limit: 500,
	}

	return s.classificationRepo.SearchClassifications(ctx, criteria)
}

// AnalyzeSuspiciousCluster analyzes suspicious transaction clusters
func (s *NodeClassificationAppService) AnalyzeSuspiciousCluster(ctx context.Context, centerAddress string, maxDepth int) (*ClusterAnalysisResult, error) {
	log.Printf("Analyzing suspicious cluster around address: %s (depth: %d)", centerAddress, maxDepth)

	// Get the cluster
	cluster, err := s.classificationRepo.GetSuspiciousCluster(ctx, centerAddress, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("failed to get suspicious cluster: %w", err)
	}

	// Analyze the cluster
	result := &ClusterAnalysisResult{
		CenterAddress:      centerAddress,
		ClusterSize:        len(cluster),
		MaxDepth:           maxDepth,
		AnalyzedAt:         time.Now(),
		Nodes:              cluster,
		RiskDistribution:   make(map[entity.NodeRiskLevel]int),
		TypeDistribution:   make(map[entity.NodeType]int),
		SuspiciousPatterns: []string{},
		Recommendations:    []string{},
	}

	// Calculate distributions
	for _, node := range cluster {
		result.RiskDistribution[node.RiskLevel]++
		result.TypeDistribution[node.PrimaryType]++

		// Analyze suspicious patterns
		if len(node.SuspiciousActivities) > 0 {
			result.SuspiciousPatterns = append(result.SuspiciousPatterns, node.SuspiciousActivities...)
		}
	}

	// Generate recommendations
	s.generateClusterRecommendations(result)

	log.Printf("Cluster analysis complete: %d nodes, %d high-risk, %d critical-risk",
		result.ClusterSize,
		result.RiskDistribution[entity.RiskLevelHigh],
		result.RiskDistribution[entity.RiskLevelCritical])

	return result, nil
}

// UpdateBlacklist adds an address to the blacklist
func (s *NodeClassificationAppService) UpdateBlacklist(ctx context.Context, address, reason string) error {
	log.Printf("Adding address to blacklist: %s (reason: %s)", address, reason)

	// Add to blacklist
	if err := s.classificationRepo.AddToBlacklist(ctx, address, reason); err != nil {
		return fmt.Errorf("failed to add to blacklist: %w", err)
	}

	// Update node classifier
	s.nodeClassifier.UpdateBlacklist(address, reason)

	// Re-classify the address
	_, err := s.ReClassifyAddress(ctx, address)
	if err != nil {
		log.Printf("Warning: failed to re-classify blacklisted address %s: %v", address, err)
	}

	return nil
}

// MonitorHighRiskTransactions monitors transactions involving high-risk addresses
func (s *NodeClassificationAppService) MonitorHighRiskTransactions(ctx context.Context) error {
	log.Println("Starting high-risk transaction monitoring")

	// Get high-risk nodes
	highRiskNodes, err := s.classificationRepo.GetHighRiskNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get high-risk nodes: %w", err)
	}

	log.Printf("Monitoring %d high-risk addresses", len(highRiskNodes))

	// For each high-risk node, check recent transactions
	for _, node := range highRiskNodes {
		connections, err := s.classificationRepo.GetNodeRelationships(ctx, node.Address)
		if err != nil {
			log.Printf("Warning: failed to get relationships for %s: %v", node.Address, err)
			continue
		}

		// Analyze connections to other high-risk nodes
		for _, conn := range connections {
			if s.isHighRiskConnection(node, conn) {
				log.Printf("High-risk connection detected: %s -> %s (type: %s)",
					conn.FromAddress, conn.ToAddress, conn.RelationshipType)
			}
		}
	}

	return nil
}

// Helper methods

func (s *NodeClassificationAppService) generateBehavioralPatterns(ctx context.Context, address string, stats *entity.WalletStats) []string {
	patterns := []string{}

	if stats == nil {
		return patterns
	}

	// High frequency pattern
	if stats.TransactionCount > 10000 {
		patterns = append(patterns, "high_frequency")
	}

	// High volume pattern
	if stats.TotalVolume != "" {
		// Add volume-based patterns
		patterns = append(patterns, "high_volume")
	}

	// Exchange-like pattern
	if stats.IncomingConnections > 1000 && stats.OutgoingConnections > 1000 {
		patterns = append(patterns, "exchange", "batch", "consolidation")
	}

	return patterns
}

func (s *NodeClassificationAppService) updateWalletWithClassification(ctx context.Context, address string, classification *entity.NodeClassification) error {
	// Update the wallet entity with classification info
	wallet, err := s.walletRepo.GetWallet(ctx, address)
	if err != nil {
		return err
	}

	if wallet == nil {
		// Create new wallet if doesn't exist
		wallet = &entity.Wallet{
			Address:   address,
			Network:   classification.Network,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
		}
	}

	// Update classification fields
	wallet.NodeType = classification.PrimaryType
	wallet.RiskLevel = classification.RiskLevel
	wallet.ConfidenceScore = classification.ConfidenceScore
	wallet.LastClassified = classification.LastClassified
	wallet.Tags = classification.Tags
	wallet.AssociatedExchanges = classification.Exchanges
	wallet.AssociatedProtocols = classification.Protocols
	wallet.IsContract = classification.PrimaryType.IsContractType()

	return s.walletRepo.CreateOrUpdateWallet(ctx, wallet)
}

func (s *NodeClassificationAppService) logClassificationChanges(address string, old, new *entity.NodeClassification) {
	if old.PrimaryType != new.PrimaryType {
		log.Printf("Classification type changed for %s: %s -> %s",
			address, old.PrimaryType, new.PrimaryType)
	}

	if old.RiskLevel != new.RiskLevel {
		log.Printf("Risk level changed for %s: %s -> %s",
			address, old.RiskLevel, new.RiskLevel)
	}

	confidenceDiff := new.ConfidenceScore - old.ConfidenceScore
	if confidenceDiff > 0.1 || confidenceDiff < -0.1 {
		log.Printf("Confidence score changed significantly for %s: %.2f -> %.2f",
			address, old.ConfidenceScore, new.ConfidenceScore)
	}
}

func (s *NodeClassificationAppService) generateClusterRecommendations(result *ClusterAnalysisResult) {
	criticalCount := result.RiskDistribution[entity.RiskLevelCritical]
	highCount := result.RiskDistribution[entity.RiskLevelHigh]

	if criticalCount > 0 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("CRITICAL: Cluster contains %d critical-risk nodes - immediate investigation required", criticalCount))
	}

	if highCount > 3 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("HIGH: Cluster contains %d high-risk nodes - enhanced monitoring recommended", highCount))
	}

	if result.ClusterSize > 20 {
		result.Recommendations = append(result.Recommendations,
			"LARGE CLUSTER: Consider deeper analysis and potential law enforcement reporting")
	}

	mixerCount := result.TypeDistribution[entity.NodeTypeMixerWallet]
	if mixerCount > 0 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("PRIVACY RISK: Cluster involves %d mixer wallets - potential money laundering", mixerCount))
	}
}

func (s *NodeClassificationAppService) isHighRiskConnection(node *entity.NodeClassification, conn *entity.NodeRelationship) bool {
	// Check if connection involves high-risk activities
	if node.RiskLevel == entity.RiskLevelCritical {
		return true
	}

	if conn.RelationshipType == string(entity.RelationshipSuspicious) ||
		conn.RelationshipType == string(entity.RelationshipBlacklisted) {
		return true
	}

	// Check connection strength and volume
	if conn.Strength > 0.8 && conn.TransactionCount > 100 {
		return true
	}

	return false
}

// ClusterAnalysisResult represents the result of a cluster analysis
type ClusterAnalysisResult struct {
	CenterAddress      string                       `json:"center_address"`
	ClusterSize        int                          `json:"cluster_size"`
	MaxDepth           int                          `json:"max_depth"`
	AnalyzedAt         time.Time                    `json:"analyzed_at"`
	Nodes              []*entity.NodeClassification `json:"nodes"`
	RiskDistribution   map[entity.NodeRiskLevel]int `json:"risk_distribution"`
	TypeDistribution   map[entity.NodeType]int      `json:"type_distribution"`
	SuspiciousPatterns []string                     `json:"suspicious_patterns"`
	Recommendations    []string                     `json:"recommendations"`
}
