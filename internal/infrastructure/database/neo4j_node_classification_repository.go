package database

import (
	"context"
	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/domain/repository"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jNodeClassificationRepository implements NodeClassificationRepository using Neo4j
type Neo4jNodeClassificationRepository struct {
	driver neo4j.DriverWithContext
}

// NewNeo4jNodeClassificationRepository creates a new Neo4j-based node classification repository
func NewNeo4jNodeClassificationRepository(driver neo4j.DriverWithContext) repository.NodeClassificationRepository {
	return &Neo4jNodeClassificationRepository{
		driver: driver,
	}
}

// CreateOrUpdateClassification creates or updates a node classification
func (r *Neo4jNodeClassificationRepository) CreateOrUpdateClassification(ctx context.Context, classification *entity.NodeClassification) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	// Convert arrays to JSON strings for Neo4j storage
	secondaryTypesJSON, _ := json.Marshal(classification.SecondaryTypes)
	detectionMethodsJSON, _ := json.Marshal(classification.DetectionMethods)
	tagsJSON, _ := json.Marshal(classification.Tags)
	exchangesJSON, _ := json.Marshal(classification.Exchanges)
	protocolsJSON, _ := json.Marshal(classification.Protocols)
	suspiciousActivitiesJSON, _ := json.Marshal(classification.SuspiciousActivities)
	blacklistReasonsJSON, _ := json.Marshal(classification.BlacklistReasons)
	sanctionDetailsJSON, _ := json.Marshal(classification.SanctionDetails)
	reportedByJSON, _ := json.Marshal(classification.ReportedBy)

	query := `
		MERGE (w:Wallet {address: $address})
		SET w.node_type = $nodeType,
			w.risk_level = $riskLevel,
			w.confidence_score = $confidenceScore,
			w.secondary_types = $secondaryTypes,
			w.detection_methods = $detectionMethods,
			w.tags = $tags,
			w.exchanges = $exchanges,
			w.protocols = $protocols,
			w.last_classified = $lastClassified,
			w.classification_count = COALESCE(w.classification_count, 0) + 1,
			w.network = $network,
			w.total_transactions = $totalTransactions,
			w.total_volume = $totalVolume,
			w.first_activity = $firstActivity,
			w.last_activity = $lastActivity,
			w.suspicious_activities = $suspiciousActivities,
			w.blacklist_reasons = $blacklistReasons,
			w.sanction_details = $sanctionDetails,
			w.reported_by = $reportedBy,
			w.is_verified = $isVerified,
			w.verification_source = $verificationSource,
			w.verified_by = $verifiedBy,
			w.verification_date = $verificationDate,
			w.updated_at = datetime()
		WITH w
		// Add classification label
		CALL apoc.create.addLabels(w, [$nodeTypeLabel]) YIELD node
		RETURN w.address as address
	`

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address":              classification.Address,
			"nodeType":             string(classification.PrimaryType),
			"nodeTypeLabel":        string(classification.PrimaryType),
			"riskLevel":            string(classification.RiskLevel),
			"confidenceScore":      classification.ConfidenceScore,
			"secondaryTypes":       string(secondaryTypesJSON),
			"detectionMethods":     string(detectionMethodsJSON),
			"tags":                 string(tagsJSON),
			"exchanges":            string(exchangesJSON),
			"protocols":            string(protocolsJSON),
			"lastClassified":       classification.LastClassified,
			"network":              classification.Network,
			"totalTransactions":    classification.TotalTransactions,
			"totalVolume":          classification.TotalVolume,
			"firstActivity":        classification.FirstActivity,
			"lastActivity":         classification.LastActivity,
			"suspiciousActivities": string(suspiciousActivitiesJSON),
			"blacklistReasons":     string(blacklistReasonsJSON),
			"sanctionDetails":      string(sanctionDetailsJSON),
			"reportedBy":           string(reportedByJSON),
			"isVerified":           classification.IsVerified,
			"verificationSource":   classification.VerificationSource,
			"verifiedBy":           classification.VerifiedBy,
			"verificationDate":     classification.VerificationDate,
		})
	})

	return err
}

// GetClassification retrieves a node classification by address
func (r *Neo4jNodeClassificationRepository) GetClassification(ctx context.Context, address string) (*entity.NodeClassification, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})
		RETURN w.address as address,
			   w.node_type as nodeType,
			   w.risk_level as riskLevel,
			   w.confidence_score as confidenceScore,
			   w.secondary_types as secondaryTypes,
			   w.detection_methods as detectionMethods,
			   w.tags as tags,
			   w.exchanges as exchanges,
			   w.protocols as protocols,
			   w.last_classified as lastClassified,
			   w.classification_count as classificationCount,
			   w.network as network,
			   w.total_transactions as totalTransactions,
			   w.total_volume as totalVolume,
			   w.first_activity as firstActivity,
			   w.last_activity as lastActivity,
			   w.suspicious_activities as suspiciousActivities,
			   w.blacklist_reasons as blacklistReasons,
			   w.sanction_details as sanctionDetails,
			   w.reported_by as reportedBy,
			   w.is_verified as isVerified,
			   w.verification_source as verificationSource,
			   w.verified_by as verifiedBy,
			   w.verification_date as verificationDate
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address": strings.ToLower(address),
		})
	})

	if err != nil {
		return nil, err
	}

	records := result.(neo4j.ResultWithContext)
	if !records.Next(ctx) {
		return nil, nil // No classification found
	}

	record := records.Record()
	return r.mapRecordToClassification(record)
}

// GetClassificationsByType retrieves all nodes of a specific type
func (r *Neo4jNodeClassificationRepository) GetClassificationsByType(ctx context.Context, nodeType entity.NodeType) ([]*entity.NodeClassification, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet)
		WHERE w.node_type = $nodeType
		RETURN w.address as address,
			   w.node_type as nodeType,
			   w.risk_level as riskLevel,
			   w.confidence_score as confidenceScore,
			   w.secondary_types as secondaryTypes,
			   w.detection_methods as detectionMethods,
			   w.tags as tags,
			   w.exchanges as exchanges,
			   w.protocols as protocols,
			   w.last_classified as lastClassified,
			   w.classification_count as classificationCount,
			   w.network as network
		ORDER BY w.confidence_score DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"nodeType": string(nodeType),
		})
	})

	if err != nil {
		return nil, err
	}

	return r.mapRecordsToClassifications(ctx, result.(neo4j.ResultWithContext))
}

// GetClassificationsByRiskLevel retrieves all nodes with a specific risk level
func (r *Neo4jNodeClassificationRepository) GetClassificationsByRiskLevel(ctx context.Context, riskLevel entity.NodeRiskLevel) ([]*entity.NodeClassification, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet)
		WHERE w.risk_level = $riskLevel
		RETURN w.address as address,
			   w.node_type as nodeType,
			   w.risk_level as riskLevel,
			   w.confidence_score as confidenceScore,
			   w.secondary_types as secondaryTypes,
			   w.detection_methods as detectionMethods,
			   w.tags as tags,
			   w.exchanges as exchanges,
			   w.protocols as protocols,
			   w.last_classified as lastClassified,
			   w.classification_count as classificationCount,
			   w.network as network
		ORDER BY w.last_classified DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"riskLevel": string(riskLevel),
		})
	})

	if err != nil {
		return nil, err
	}

	return r.mapRecordsToClassifications(ctx, result.(neo4j.ResultWithContext))
}

// SearchClassifications searches for classifications based on criteria
func (r *Neo4jNodeClassificationRepository) SearchClassifications(ctx context.Context, criteria *repository.ClassificationSearchCriteria) ([]*entity.NodeClassification, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Build dynamic query based on criteria
	whereConditions := []string{}
	params := map[string]interface{}{}

	if len(criteria.NodeTypes) > 0 {
		nodeTypes := make([]string, len(criteria.NodeTypes))
		for i, nt := range criteria.NodeTypes {
			nodeTypes[i] = string(nt)
		}
		whereConditions = append(whereConditions, "w.node_type IN $nodeTypes")
		params["nodeTypes"] = nodeTypes
	}

	if len(criteria.RiskLevels) > 0 {
		riskLevels := make([]string, len(criteria.RiskLevels))
		for i, rl := range criteria.RiskLevels {
			riskLevels[i] = string(rl)
		}
		whereConditions = append(whereConditions, "w.risk_level IN $riskLevels")
		params["riskLevels"] = riskLevels
	}

	if criteria.MinConfidenceScore > 0 {
		whereConditions = append(whereConditions, "w.confidence_score >= $minConfidence")
		params["minConfidence"] = criteria.MinConfidenceScore
	}

	if criteria.HasSuspiciousActivity {
		whereConditions = append(whereConditions, "w.suspicious_activities IS NOT NULL AND w.suspicious_activities <> '[]'")
	}

	if criteria.IsBlacklisted {
		whereConditions = append(whereConditions, "w.blacklist_reasons IS NOT NULL AND w.blacklist_reasons <> '[]'")
	}

	if criteria.IsSanctioned {
		whereConditions = append(whereConditions, "w.sanction_details IS NOT NULL AND w.sanction_details <> '[]'")
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	query := fmt.Sprintf(`
		MATCH (w:Wallet)
		%s
		RETURN w.address as address,
			   w.node_type as nodeType,
			   w.risk_level as riskLevel,
			   w.confidence_score as confidenceScore,
			   w.secondary_types as secondaryTypes,
			   w.detection_methods as detectionMethods,
			   w.tags as tags,
			   w.exchanges as exchanges,
			   w.protocols as protocols,
			   w.last_classified as lastClassified,
			   w.classification_count as classificationCount,
			   w.network as network
		ORDER BY w.confidence_score DESC
		SKIP $offset
		LIMIT $limit
	`, whereClause)

	params["offset"] = criteria.Offset
	params["limit"] = criteria.Limit
	if criteria.Limit == 0 {
		params["limit"] = 100 // Default limit
	}

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, params)
	})

	if err != nil {
		return nil, err
	}

	return r.mapRecordsToClassifications(ctx, result.(neo4j.ResultWithContext))
}

// UpdateRiskLevel updates the risk level for a specific address
func (r *Neo4jNodeClassificationRepository) UpdateRiskLevel(ctx context.Context, address string, riskLevel entity.NodeRiskLevel, reason string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})
		SET w.risk_level = $riskLevel,
			w.risk_update_reason = $reason,
			w.risk_updated_at = datetime()
		RETURN w.address as address
	`

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address":   strings.ToLower(address),
			"riskLevel": string(riskLevel),
			"reason":    reason,
		})
	})

	return err
}

// AddToBlacklist adds an address to the blacklist
func (r *Neo4jNodeClassificationRepository) AddToBlacklist(ctx context.Context, address, reason string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := `
		MERGE (w:Wallet {address: $address})
		SET w.is_blacklisted = true,
			w.blacklist_reason = $reason,
			w.blacklisted_at = datetime(),
			w.risk_level = 'CRITICAL'
		WITH w
		MERGE (b:Blacklist {address: $address})
		SET b.reason = $reason,
			b.added_at = datetime()
		MERGE (w)-[:BLACKLISTED]->(b)
		RETURN w.address as address
	`

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address": strings.ToLower(address),
			"reason":  reason,
		})
	})

	return err
}

// RemoveFromBlacklist removes an address from the blacklist
func (r *Neo4jNodeClassificationRepository) RemoveFromBlacklist(ctx context.Context, address string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})
		REMOVE w.is_blacklisted, w.blacklist_reason, w.blacklisted_at
		WITH w
		MATCH (w)-[r:BLACKLISTED]->(b:Blacklist {address: $address})
		DELETE r, b
		RETURN w.address as address
	`

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address": strings.ToLower(address),
		})
	})

	return err
}

// GetBlacklistedAddresses retrieves all blacklisted addresses
func (r *Neo4jNodeClassificationRepository) GetBlacklistedAddresses(ctx context.Context) (map[string]string, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (b:Blacklist)
		RETURN b.address as address, b.reason as reason
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{})
	})

	if err != nil {
		return nil, err
	}

	blacklist := make(map[string]string)
	records := result.(neo4j.ResultWithContext)
	for records.Next(ctx) {
		record := records.Record()
		address, _ := record.Get("address")
		reason, _ := record.Get("reason")
		blacklist[address.(string)] = reason.(string)
	}

	return blacklist, nil
}

// CreateNodeRelationship creates a relationship between two nodes
func (r *Neo4jNodeClassificationRepository) CreateNodeRelationship(ctx context.Context, relationship *entity.NodeRelationship) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	// Convert properties to JSON
	propertiesJSON, _ := json.Marshal(relationship.Properties)

	query := fmt.Sprintf(`
		MATCH (from:Wallet {address: $fromAddress})
		MATCH (to:Wallet {address: $toAddress})
		MERGE (from)-[r:%s]->(to)
		SET r.strength = $strength,
			r.total_value = $totalValue,
			r.transaction_count = $transactionCount,
			r.first_seen = $firstSeen,
			r.last_seen = $lastSeen,
			r.network = $network,
			r.confidence = $confidence,
			r.detection_method = $detectionMethod,
			r.properties = $properties,
			r.updated_at = datetime()
		RETURN r
	`, relationship.RelationshipType)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"fromAddress":      strings.ToLower(relationship.FromAddress),
			"toAddress":        strings.ToLower(relationship.ToAddress),
			"strength":         relationship.Strength,
			"totalValue":       relationship.TotalValue,
			"transactionCount": relationship.TransactionCount,
			"firstSeen":        relationship.FirstSeen,
			"lastSeen":         relationship.LastSeen,
			"network":          relationship.Network,
			"confidence":       relationship.Confidence,
			"detectionMethod":  relationship.DetectionMethod,
			"properties":       string(propertiesJSON),
		})
	})

	return err
}

// GetNodeRelationships retrieves relationships for a specific address
func (r *Neo4jNodeClassificationRepository) GetNodeRelationships(ctx context.Context, address string) ([]*entity.NodeRelationship, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})-[r]-(other:Wallet)
		RETURN w.address as fromAddress,
			   other.address as toAddress,
			   type(r) as relationshipType,
			   r.strength as strength,
			   r.total_value as totalValue,
			   r.transaction_count as transactionCount,
			   r.first_seen as firstSeen,
			   r.last_seen as lastSeen,
			   r.network as network,
			   r.confidence as confidence,
			   r.detection_method as detectionMethod,
			   r.properties as properties
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address": strings.ToLower(address),
		})
	})

	if err != nil {
		return nil, err
	}

	relationships := []*entity.NodeRelationship{}
	records := result.(neo4j.ResultWithContext)
	for records.Next(ctx) {
		record := records.Record()

		var properties map[string]interface{}
		if propStr, ok := record.Get("properties"); ok && propStr != nil {
			json.Unmarshal([]byte(propStr.(string)), &properties)
		}

		relationship := &entity.NodeRelationship{
			FromAddress:      getString(record, "fromAddress"),
			ToAddress:        getString(record, "toAddress"),
			RelationshipType: getString(record, "relationshipType"),
			Strength:         getFloat64(record, "strength"),
			TotalValue:       getString(record, "totalValue"),
			TransactionCount: getInt64(record, "transactionCount"),
			FirstSeen:        getTime(record, "firstSeen"),
			LastSeen:         getTime(record, "lastSeen"),
			Network:          getString(record, "network"),
			Confidence:       getFloat64(record, "confidence"),
			DetectionMethod:  getString(record, "detectionMethod"),
			Properties:       properties,
		}
		relationships = append(relationships, relationship)
	}

	return relationships, nil
}

// GetSuspiciousCluster identifies clusters of suspicious nodes
func (r *Neo4jNodeClassificationRepository) GetSuspiciousCluster(ctx context.Context, address string, maxDepth int) ([]*entity.NodeClassification, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH path = (start:Wallet {address: $address})-[*1..%d]-(connected:Wallet)
		WHERE connected.risk_level IN ['HIGH', 'CRITICAL']
		   OR connected.is_blacklisted = true
		   OR connected.suspicious_activities IS NOT NULL
		RETURN DISTINCT connected.address as address,
			   connected.node_type as nodeType,
			   connected.risk_level as riskLevel,
			   connected.confidence_score as confidenceScore,
			   connected.secondary_types as secondaryTypes,
			   connected.detection_methods as detectionMethods,
			   connected.tags as tags,
			   connected.exchanges as exchanges,
			   connected.protocols as protocols,
			   connected.last_classified as lastClassified,
			   connected.classification_count as classificationCount,
			   connected.network as network
		ORDER BY connected.risk_level DESC, connected.confidence_score DESC
		LIMIT 50
	`, maxDepth)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address": strings.ToLower(address),
		})
	})

	if err != nil {
		return nil, err
	}

	return r.mapRecordsToClassifications(ctx, result.(neo4j.ResultWithContext))
}

// GetExchangeWallets retrieves all wallets associated with a specific exchange
func (r *Neo4jNodeClassificationRepository) GetExchangeWallets(ctx context.Context, exchange string) ([]*entity.NodeClassification, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet)
		WHERE w.exchanges CONTAINS $exchange
		RETURN w.address as address,
			   w.node_type as nodeType,
			   w.risk_level as riskLevel,
			   w.confidence_score as confidenceScore,
			   w.secondary_types as secondaryTypes,
			   w.detection_methods as detectionMethods,
			   w.tags as tags,
			   w.exchanges as exchanges,
			   w.protocols as protocols,
			   w.last_classified as lastClassified,
			   w.classification_count as classificationCount,
			   w.network as network
		ORDER BY w.last_classified DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"exchange": exchange,
		})
	})

	if err != nil {
		return nil, err
	}

	return r.mapRecordsToClassifications(ctx, result.(neo4j.ResultWithContext))
}

// GetHighRiskNodes retrieves all high-risk and critical nodes
func (r *Neo4jNodeClassificationRepository) GetHighRiskNodes(ctx context.Context) ([]*entity.NodeClassification, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet)
		WHERE w.risk_level IN ['HIGH', 'CRITICAL']
		RETURN w.address as address,
			   w.node_type as nodeType,
			   w.risk_level as riskLevel,
			   w.confidence_score as confidenceScore,
			   w.secondary_types as secondaryTypes,
			   w.detection_methods as detectionMethods,
			   w.tags as tags,
			   w.exchanges as exchanges,
			   w.protocols as protocols,
			   w.last_classified as lastClassified,
			   w.classification_count as classificationCount,
			   w.network as network
		ORDER BY
			CASE w.risk_level
				WHEN 'CRITICAL' THEN 1
				WHEN 'HIGH' THEN 2
			END,
			w.confidence_score DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{})
	})

	if err != nil {
		return nil, err
	}

	return r.mapRecordsToClassifications(ctx, result.(neo4j.ResultWithContext))
}

// UpdateClassificationStats updates classification statistics
func (r *Neo4jNodeClassificationRepository) UpdateClassificationStats(ctx context.Context, address string, stats *entity.WalletStats) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := `
		MATCH (w:Wallet {address: $address})
		SET w.total_transactions = $totalTransactions,
			w.total_volume = $totalVolume,
			w.incoming_connections = $incomingConnections,
			w.outgoing_connections = $outgoingConnections,
			w.stats_updated_at = datetime()
		RETURN w.address as address
	`

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address":             strings.ToLower(address),
			"totalTransactions":   stats.TransactionCount,
			"totalVolume":         stats.TotalVolume,
			"incomingConnections": stats.IncomingConnections,
			"outgoingConnections": stats.OutgoingConnections,
		})
	})

	return err
}

// GetClassificationHistory retrieves the classification history for an address
func (r *Neo4jNodeClassificationRepository) GetClassificationHistory(ctx context.Context, address string) ([]*repository.ClassificationHistoryEntry, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (h:ClassificationHistory {address: $address})
		RETURN h.address as address,
			   h.previous_type as previousType,
			   h.new_type as newType,
			   h.previous_risk_level as previousRiskLevel,
			   h.new_risk_level as newRiskLevel,
			   h.confidence_score as confidenceScore,
			   h.detection_method as detectionMethod,
			   h.reason as reason,
			   h.timestamp as timestamp,
			   h.classified_by as classifiedBy
		ORDER BY h.timestamp DESC
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"address": strings.ToLower(address),
		})
	})

	if err != nil {
		return nil, err
	}

	history := []*repository.ClassificationHistoryEntry{}
	records := result.(neo4j.ResultWithContext)
	for records.Next(ctx) {
		record := records.Record()
		entry := &repository.ClassificationHistoryEntry{
			Address:           getString(record, "address"),
			PreviousType:      entity.NodeType(getString(record, "previousType")),
			NewType:           entity.NodeType(getString(record, "newType")),
			PreviousRiskLevel: entity.NodeRiskLevel(getString(record, "previousRiskLevel")),
			NewRiskLevel:      entity.NodeRiskLevel(getString(record, "newRiskLevel")),
			ConfidenceScore:   getFloat64(record, "confidenceScore"),
			DetectionMethod:   getString(record, "detectionMethod"),
			Reason:            getString(record, "reason"),
			Timestamp:         getString(record, "timestamp"),
			ClassifiedBy:      getString(record, "classifiedBy"),
		}
		history = append(history, entry)
	}

	return history, nil
}

// BulkUpdateClassifications updates multiple classifications in a batch
func (r *Neo4jNodeClassificationRepository) BulkUpdateClassifications(ctx context.Context, classifications []*entity.NodeClassification) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for _, classification := range classifications {
			secondaryTypesJSON, _ := json.Marshal(classification.SecondaryTypes)
			detectionMethodsJSON, _ := json.Marshal(classification.DetectionMethods)
			tagsJSON, _ := json.Marshal(classification.Tags)

			query := `
				MERGE (w:Wallet {address: $address})
				SET w.node_type = $nodeType,
					w.risk_level = $riskLevel,
					w.confidence_score = $confidenceScore,
					w.secondary_types = $secondaryTypes,
					w.detection_methods = $detectionMethods,
					w.tags = $tags,
					w.last_classified = $lastClassified,
					w.classification_count = COALESCE(w.classification_count, 0) + 1,
					w.network = $network,
					w.updated_at = datetime()
			`

			_, err := tx.Run(ctx, query, map[string]interface{}{
				"address":          classification.Address,
				"nodeType":         string(classification.PrimaryType),
				"riskLevel":        string(classification.RiskLevel),
				"confidenceScore":  classification.ConfidenceScore,
				"secondaryTypes":   string(secondaryTypesJSON),
				"detectionMethods": string(detectionMethodsJSON),
				"tags":             string(tagsJSON),
				"lastClassified":   classification.LastClassified,
				"network":          classification.Network,
			})

			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	return err
}

// Helper methods for mapping Neo4j records to entities

func (r *Neo4jNodeClassificationRepository) mapRecordToClassification(record *neo4j.Record) (*entity.NodeClassification, error) {
	classification := &entity.NodeClassification{}

	classification.Address = getString(record, "address")
	classification.PrimaryType = entity.NodeType(getString(record, "nodeType"))
	classification.RiskLevel = entity.NodeRiskLevel(getString(record, "riskLevel"))
	classification.ConfidenceScore = getFloat64(record, "confidenceScore")
	classification.LastClassified = getTime(record, "lastClassified")
	classification.ClassificationCount = getInt64(record, "classificationCount")
	classification.Network = getString(record, "network")
	classification.TotalTransactions = getInt64(record, "totalTransactions")
	classification.TotalVolume = getString(record, "totalVolume")
	classification.FirstActivity = getTime(record, "firstActivity")
	classification.LastActivity = getTime(record, "lastActivity")
	classification.IsVerified = getBool(record, "isVerified")
	classification.VerificationSource = getString(record, "verificationSource")
	classification.VerifiedBy = getString(record, "verifiedBy")
	classification.VerificationDate = getTime(record, "verificationDate")

	// Parse JSON arrays
	if secondaryTypesStr := getString(record, "secondaryTypes"); secondaryTypesStr != "" {
		json.Unmarshal([]byte(secondaryTypesStr), &classification.SecondaryTypes)
	}

	if detectionMethodsStr := getString(record, "detectionMethods"); detectionMethodsStr != "" {
		json.Unmarshal([]byte(detectionMethodsStr), &classification.DetectionMethods)
	}

	if tagsStr := getString(record, "tags"); tagsStr != "" {
		json.Unmarshal([]byte(tagsStr), &classification.Tags)
	}

	if exchangesStr := getString(record, "exchanges"); exchangesStr != "" {
		json.Unmarshal([]byte(exchangesStr), &classification.Exchanges)
	}

	if protocolsStr := getString(record, "protocols"); protocolsStr != "" {
		json.Unmarshal([]byte(protocolsStr), &classification.Protocols)
	}

	if suspiciousStr := getString(record, "suspiciousActivities"); suspiciousStr != "" {
		json.Unmarshal([]byte(suspiciousStr), &classification.SuspiciousActivities)
	}

	if blacklistStr := getString(record, "blacklistReasons"); blacklistStr != "" {
		json.Unmarshal([]byte(blacklistStr), &classification.BlacklistReasons)
	}

	if sanctionStr := getString(record, "sanctionDetails"); sanctionStr != "" {
		json.Unmarshal([]byte(sanctionStr), &classification.SanctionDetails)
	}

	if reportedByStr := getString(record, "reportedBy"); reportedByStr != "" {
		json.Unmarshal([]byte(reportedByStr), &classification.ReportedBy)
	}

	return classification, nil
}

func (r *Neo4jNodeClassificationRepository) mapRecordsToClassifications(ctx context.Context, records neo4j.ResultWithContext) ([]*entity.NodeClassification, error) {
	classifications := []*entity.NodeClassification{}

	for records.Next(ctx) {
		record := records.Record()
		classification, err := r.mapRecordToClassification(record)
		if err != nil {
			return nil, err
		}
		classifications = append(classifications, classification)
	}

	return classifications, nil
}

// Helper functions to safely extract values from Neo4j records
func getString(record *neo4j.Record, key string) string {
	if val, ok := record.Get(key); ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloat64(record *neo4j.Record, key string) float64 {
	if val, ok := record.Get(key); ok && val != nil {
		if f, ok := val.(float64); ok {
			return f
		}
		if i, ok := val.(int64); ok {
			return float64(i)
		}
	}
	return 0.0
}

func getInt64(record *neo4j.Record, key string) int64 {
	if val, ok := record.Get(key); ok && val != nil {
		if i, ok := val.(int64); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int64(f)
		}
	}
	return 0
}

func getBool(record *neo4j.Record, key string) bool {
	if val, ok := record.Get(key); ok && val != nil {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getTime(record *neo4j.Record, key string) time.Time {
	if val, ok := record.Get(key); ok && val != nil {
		if t, ok := val.(time.Time); ok {
			return t
		}
		if str, ok := val.(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}
