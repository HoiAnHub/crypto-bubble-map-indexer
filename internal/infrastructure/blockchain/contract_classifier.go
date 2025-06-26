package blockchain

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/domain/service"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ContractClassifierService implements contract classification logic
type ContractClassifierService struct {
	logger              *logger.Logger
	classificationRules []entity.ClassificationRule
}

// NewContractClassifierService creates a new contract classifier service
func NewContractClassifierService(logger *logger.Logger) service.ContractClassifierService {
	classifier := &ContractClassifierService{
		logger: logger.WithComponent("contract-classifier"),
	}
	classifier.initializeClassificationRules()
	return classifier
}

// ClassifyContract analyzes contract interactions to determine contract type
func (c *ContractClassifierService) ClassifyContract(ctx context.Context, contractAddress string, interactions []*entity.ERC20Transfer) (*entity.ContractClassification, error) {
	if len(interactions) == 0 {
		return c.createBasicClassification(contractAddress), nil
	}

	// Analyze method signatures and interaction patterns
	methodCounts := make(map[string]int)
	interactionPatterns := make(map[entity.ContractInteractionType]int)
	uniqueUsers := make(map[string]bool)
	var firstSeen, lastSeen time.Time

	for i, interaction := range interactions {
		// Count method signatures
		methodCounts[interaction.MethodSignature]++

		// Count interaction patterns
		interactionPatterns[interaction.InteractionType]++

		// Track unique users
		uniqueUsers[interaction.From] = true

		// Track time range
		if i == 0 {
			firstSeen = interaction.Timestamp
			lastSeen = interaction.Timestamp
		} else {
			if interaction.Timestamp.Before(firstSeen) {
				firstSeen = interaction.Timestamp
			}
			if interaction.Timestamp.After(lastSeen) {
				lastSeen = interaction.Timestamp
			}
		}
	}

	// Apply classification rules
	classification := &entity.ContractClassification{
		Address:             contractAddress,
		MethodSignatures:    methodCounts,
		InteractionPatterns: interactionPatterns,
		TotalInteractions:   int64(len(interactions)),
		UniqueUsers:         int64(len(uniqueUsers)),
		FirstSeen:           firstSeen,
		LastSeen:            lastSeen,
		Network:             interactions[0].Network,
	}

	// Determine primary and secondary types
	c.classifyByRules(classification)

	// Add detected protocols
	c.detectProtocols(classification)

	// Add tags based on activity
	c.addActivityTags(classification)

	c.logger.Info("Contract classified",
		zap.String("address", contractAddress),
		zap.String("primary_type", string(classification.PrimaryType)),
		zap.Float64("confidence", classification.ConfidenceScore),
		zap.Strings("protocols", classification.DetectedProtocols),
		zap.Int64("total_interactions", classification.TotalInteractions),
		zap.Int64("unique_users", classification.UniqueUsers))

	return classification, nil
}

// ClassifyFromMethodSignature provides quick classification based on method signature
func (c *ContractClassifierService) ClassifyFromMethodSignature(methodSignature string) entity.ContractType {
	methodSig := strings.ToLower(methodSignature)

	switch methodSig {
	// DEX/AMM Signatures
	case "7ff36ab5", "18cbafe5", "38ed1739": // Uniswap swaps
		return entity.ContractTypeDEX
	case "022c0d9f": // Uniswap V2 swap
		return entity.ContractTypeUniswapV2
	case "e8e33700", "baa2abde": // Add/Remove liquidity
		return entity.ContractTypeAMM

	// Lending Protocol Signatures
	case "d0e30db0", "2e1a7d4d": // deposit, withdraw (common)
		return entity.ContractTypeLendingPool
	case "a6afed95", "852a12e3": // Compound mint, redeem
		return entity.ContractTypeCompound
	case "d65d7f80", "69328dec", "e8eda9df": // Aave specific
		return entity.ContractTypeAave

	// Utility Contracts
	case "ac9650d8", "5ae401dc": // Multicall
		return entity.ContractTypeMulticall

	// Standard ERC20
	case "a9059cbb", "23b872dd", "095ea7b3": // transfer, transferFrom, approve
		return entity.ContractTypeERC20

	default:
		return entity.ContractTypeUnknown
	}
}

// UpdateContractClassification updates classification based on new interactions
func (c *ContractClassifierService) UpdateContractClassification(ctx context.Context, classification *entity.ContractClassification, newInteraction *entity.ERC20Transfer) *entity.ContractClassification {
	// Update method signature counts
	if classification.MethodSignatures == nil {
		classification.MethodSignatures = make(map[string]int)
	}
	classification.MethodSignatures[newInteraction.MethodSignature]++

	// Update interaction patterns
	if classification.InteractionPatterns == nil {
		classification.InteractionPatterns = make(map[entity.ContractInteractionType]int)
	}
	classification.InteractionPatterns[newInteraction.InteractionType]++

	// Update counters
	classification.TotalInteractions++
	classification.LastSeen = newInteraction.Timestamp

	// Re-classify if we have enough new data
	if classification.TotalInteractions%100 == 0 { // Re-classify every 100 interactions
		c.classifyByRules(classification)
		c.detectProtocols(classification)
		c.addActivityTags(classification)
	}

	return classification
}

// GetContractPatterns returns known patterns for contract types
func (c *ContractClassifierService) GetContractPatterns() map[entity.ContractType][]string {
	patterns := make(map[entity.ContractType][]string)

	for _, rule := range c.classificationRules {
		patterns[rule.ContractType] = append(rule.RequiredMethods, rule.OptionalMethods...)
	}

	return patterns
}

// initializeClassificationRules sets up classification rules
func (c *ContractClassifierService) initializeClassificationRules() {
	c.classificationRules = []entity.ClassificationRule{
		// Uniswap V2 Router
		{
			ContractType:        entity.ContractTypeUniswapV2,
			RequiredMethods:     []string{"7ff36ab5", "18cbafe5"},             // swapExactETHForTokens, swapExactTokensForETH
			OptionalMethods:     []string{"38ed1739", "e8e33700", "baa2abde"}, // swapTokens, addLiquidity, removeLiquidity
			InteractionPatterns: []entity.ContractInteractionType{entity.InteractionSwap, entity.InteractionAddLiquidity},
			MinConfidence:       0.8,
			Weight:              1.0,
		},
		// Generic DEX/AMM
		{
			ContractType:        entity.ContractTypeDEX,
			RequiredMethods:     []string{"7ff36ab5"}, // Any swap method
			OptionalMethods:     []string{"18cbafe5", "38ed1739"},
			InteractionPatterns: []entity.ContractInteractionType{entity.InteractionSwap},
			MinConfidence:       0.6,
			Weight:              0.8,
		},
		// Compound Protocol
		{
			ContractType:        entity.ContractTypeCompound,
			RequiredMethods:     []string{"a6afed95"},                         // mint
			OptionalMethods:     []string{"852a12e3", "d0e30db0", "2e1a7d4d"}, // redeem, deposit, withdraw
			InteractionPatterns: []entity.ContractInteractionType{entity.InteractionDeposit, entity.InteractionWithdraw},
			MinConfidence:       0.7,
			Weight:              0.9,
		},
		// AAVE Protocol
		{
			ContractType:        entity.ContractTypeAave,
			RequiredMethods:     []string{"d65d7f80"},             // deposit (Aave specific)
			OptionalMethods:     []string{"69328dec", "e8eda9df"}, // withdraw, borrow
			InteractionPatterns: []entity.ContractInteractionType{entity.InteractionDeposit, entity.InteractionWithdraw},
			MinConfidence:       0.8,
			Weight:              0.9,
		},
		// Generic Lending Pool
		{
			ContractType:        entity.ContractTypeLendingPool,
			RequiredMethods:     []string{"d0e30db0", "2e1a7d4d"}, // deposit, withdraw
			InteractionPatterns: []entity.ContractInteractionType{entity.InteractionDeposit, entity.InteractionWithdraw},
			MinConfidence:       0.6,
			Weight:              0.7,
		},
		// WETH Contract
		{
			ContractType:    entity.ContractTypeWETH,
			RequiredMethods: []string{"d0e30db0", "2e1a7d4d"}, // deposit, withdraw
			ExcludeMethods:  []string{"7ff36ab5", "a6afed95"}, // Not swap or mint
			MinConfidence:   0.9,
			Weight:          1.0,
		},
		// Multicall Contract
		{
			ContractType:        entity.ContractTypeMulticall,
			RequiredMethods:     []string{"ac9650d8"}, // multicall
			OptionalMethods:     []string{"5ae401dc"}, // multicallWithDeadline
			InteractionPatterns: []entity.ContractInteractionType{entity.InteractionMulticall},
			MinConfidence:       0.9,
			Weight:              1.0,
		},
		// Standard ERC20 Token
		{
			ContractType:        entity.ContractTypeERC20,
			RequiredMethods:     []string{"a9059cbb"},             // transfer
			OptionalMethods:     []string{"23b872dd", "095ea7b3"}, // transferFrom, approve
			InteractionPatterns: []entity.ContractInteractionType{entity.InteractionTransfer, entity.InteractionApprove},
			MinConfidence:       0.5,
			Weight:              0.6,
		},
	}
}

// classifyByRules applies classification rules to determine contract type
func (c *ContractClassifierService) classifyByRules(classification *entity.ContractClassification) {
	scores := make(map[entity.ContractType]float64)

	for _, rule := range c.classificationRules {
		score := c.calculateRuleScore(classification, rule)
		if score >= rule.MinConfidence {
			scores[rule.ContractType] = score * rule.Weight
		}
	}

	// Find primary type (highest score)
	var primaryType entity.ContractType = entity.ContractTypeUnknown
	var maxScore float64 = 0
	var secondaryTypes []entity.ContractType

	for contractType, score := range scores {
		if score > maxScore {
			if primaryType != entity.ContractTypeUnknown {
				secondaryTypes = append(secondaryTypes, primaryType)
			}
			primaryType = contractType
			maxScore = score
		} else if score > 0.5 { // Secondary type threshold
			secondaryTypes = append(secondaryTypes, contractType)
		}
	}

	classification.PrimaryType = primaryType
	classification.SecondaryTypes = secondaryTypes
	classification.ConfidenceScore = maxScore
}

// calculateRuleScore calculates how well a classification matches a rule
func (c *ContractClassifierService) calculateRuleScore(classification *entity.ContractClassification, rule entity.ClassificationRule) float64 {
	score := 0.0
	totalChecks := 0

	// Check required methods (must have all)
	requiredScore := 0.0
	for _, method := range rule.RequiredMethods {
		totalChecks++
		if count, exists := classification.MethodSignatures[method]; exists && count > 0 {
			requiredScore += 1.0
		}
	}
	if len(rule.RequiredMethods) > 0 {
		score += (requiredScore / float64(len(rule.RequiredMethods))) * 0.6 // 60% weight for required
	}

	// Check optional methods (nice to have)
	optionalScore := 0.0
	for _, method := range rule.OptionalMethods {
		totalChecks++
		if count, exists := classification.MethodSignatures[method]; exists && count > 0 {
			optionalScore += 1.0
		}
	}
	if len(rule.OptionalMethods) > 0 {
		score += (optionalScore / float64(len(rule.OptionalMethods))) * 0.2 // 20% weight for optional
	}

	// Check excluded methods (must not have)
	excludeScore := 1.0
	for _, method := range rule.ExcludeMethods {
		if count, exists := classification.MethodSignatures[method]; exists && count > 0 {
			excludeScore = 0.0 // Fails if any excluded method is found
			break
		}
	}
	score += excludeScore * 0.1 // 10% weight for exclusions

	// Check interaction patterns
	patternScore := 0.0
	for _, pattern := range rule.InteractionPatterns {
		if count, exists := classification.InteractionPatterns[pattern]; exists && count > 0 {
			patternScore += 1.0
		}
	}
	if len(rule.InteractionPatterns) > 0 {
		score += (patternScore / float64(len(rule.InteractionPatterns))) * 0.1 // 10% weight for patterns
	}

	return math.Min(score, 1.0) // Cap at 1.0
}

// detectProtocols identifies specific protocols based on method signatures
func (c *ContractClassifierService) detectProtocols(classification *entity.ContractClassification) {
	var protocols []string

	// Check for specific protocol signatures
	if c.hasAnyMethod(classification.MethodSignatures, []string{"7ff36ab5", "18cbafe5", "022c0d9f"}) {
		protocols = append(protocols, "uniswap")
	}
	if c.hasAnyMethod(classification.MethodSignatures, []string{"a6afed95", "852a12e3"}) {
		protocols = append(protocols, "compound")
	}
	if c.hasAnyMethod(classification.MethodSignatures, []string{"d65d7f80", "69328dec"}) {
		protocols = append(protocols, "aave")
	}
	if c.hasAnyMethod(classification.MethodSignatures, []string{"ac9650d8"}) {
		protocols = append(protocols, "multicall")
	}

	classification.DetectedProtocols = protocols
}

// addActivityTags adds tags based on contract activity
func (c *ContractClassifierService) addActivityTags(classification *entity.ContractClassification) {
	var tags []string

	// Activity level tags
	if classification.TotalInteractions > 10000 {
		tags = append(tags, "high-volume")
	} else if classification.TotalInteractions > 1000 {
		tags = append(tags, "medium-volume")
	} else {
		tags = append(tags, "low-volume")
	}

	// User engagement tags
	if classification.UniqueUsers > 1000 {
		tags = append(tags, "popular")
	}

	// Protocol type tags
	if classification.PrimaryType.IsDefiProtocol() {
		tags = append(tags, "defi")
	}

	category := classification.PrimaryType.GetContractTypeCategory()
	if category != "OTHER" {
		tags = append(tags, strings.ToLower(category))
	}

	classification.Tags = tags
}

// createBasicClassification creates a basic classification for new contracts
func (c *ContractClassifierService) createBasicClassification(contractAddress string) *entity.ContractClassification {
	return &entity.ContractClassification{
		Address:             contractAddress,
		PrimaryType:         entity.ContractTypeUnknown,
		SecondaryTypes:      []entity.ContractType{},
		ConfidenceScore:     0.0,
		DetectedProtocols:   []string{},
		MethodSignatures:    make(map[string]int),
		InteractionPatterns: make(map[entity.ContractInteractionType]int),
		TotalInteractions:   0,
		UniqueUsers:         0,
		FirstSeen:           time.Now(),
		LastSeen:            time.Now(),
		IsVerified:          false,
		VerificationSource:  "heuristic",
		Tags:                []string{"new"},
	}
}

// hasAnyMethod checks if any of the specified methods exist in the method signatures
func (c *ContractClassifierService) hasAnyMethod(methodSignatures map[string]int, methods []string) bool {
	for _, method := range methods {
		if count, exists := methodSignatures[method]; exists && count > 0 {
			return true
		}
	}
	return false
}
