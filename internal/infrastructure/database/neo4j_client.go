package database

import (
	"context"
	"fmt"

	"crypto-bubble-map-indexer/internal/infrastructure/config"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/zap"
)

// Neo4JClient handles Neo4J database operations
type Neo4JClient struct {
	driver neo4j.DriverWithContext
	config *config.Neo4JConfig
	logger *logger.Logger
}

// NewNeo4JClient creates a new Neo4J client
func NewNeo4JClient(cfg *config.Neo4JConfig, logger *logger.Logger) *Neo4JClient {
	return &Neo4JClient{
		config: cfg,
		logger: logger.WithComponent("neo4j-client"),
	}
}

// Connect connects to Neo4J database
func (n *Neo4JClient) Connect(ctx context.Context) error {
	n.logger.Info("Connecting to Neo4J database", zap.String("uri", n.config.URI))

	driver, err := neo4j.NewDriverWithContext(
		n.config.URI,
		neo4j.BasicAuth(n.config.Username, n.config.Password, ""),
		func(config *neo4j.Config) {
			config.MaxConnectionPoolSize = n.config.MaxConnectionPoolSize
			config.ConnectionAcquisitionTimeout = n.config.ConnectionAcquisitionTimeout
		},
	)
	if err != nil {
		n.logger.Error("Failed to create Neo4J driver", zap.Error(err))
		return fmt.Errorf("failed to create Neo4J driver: %w", err)
	}

	// Verify connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		n.logger.Error("Failed to verify Neo4J connectivity", zap.Error(err))
		return fmt.Errorf("failed to verify Neo4J connectivity: %w", err)
	}

	n.driver = driver
	n.logger.Info("Successfully connected to Neo4J database")

	// Setup database schema
	if err := n.setupSchema(ctx); err != nil {
		return fmt.Errorf("failed to setup schema: %w", err)
	}

	return nil
}

// Close closes the Neo4J connection
func (n *Neo4JClient) Close(ctx context.Context) error {
	if n.driver != nil {
		n.logger.Info("Closing Neo4J connection")
		return n.driver.Close(ctx)
	}
	return nil
}

// GetDriver returns the Neo4J driver
func (n *Neo4JClient) GetDriver() neo4j.DriverWithContext {
	return n.driver
}

// setupSchema creates the necessary constraints and indexes
func (n *Neo4JClient) setupSchema(ctx context.Context) error {
	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
	})
	defer session.Close(ctx)

	// Create constraints
	constraints := []string{
		"CREATE CONSTRAINT wallet_address IF NOT EXISTS FOR (w:Wallet) REQUIRE w.address IS UNIQUE",
		"CREATE CONSTRAINT transaction_hash IF NOT EXISTS FOR (t:Transaction) REQUIRE t.hash IS UNIQUE",
	}

	for _, constraint := range constraints {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			return tx.Run(ctx, constraint, nil)
		})
		if err != nil {
			n.logger.Warn("Failed to create constraint", zap.String("constraint", constraint), zap.Error(err))
		}
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX wallet_first_seen IF NOT EXISTS FOR (w:Wallet) ON (w.first_seen)",
		"CREATE INDEX wallet_last_seen IF NOT EXISTS FOR (w:Wallet) ON (w.last_seen)",
		"CREATE INDEX transaction_timestamp IF NOT EXISTS FOR (t:Transaction) ON (t.timestamp)",
		"CREATE INDEX transaction_block_number IF NOT EXISTS FOR (t:Transaction) ON (t.block_number)",
	}

	for _, index := range indexes {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			return tx.Run(ctx, index, nil)
		})
		if err != nil {
			n.logger.Warn("Failed to create index", zap.String("index", index), zap.Error(err))
		}
	}

	n.logger.Info("Schema setup completed")
	return nil
}

// IsConnected checks if connected to Neo4J
func (n *Neo4JClient) IsConnected(ctx context.Context) bool {
	if n.driver == nil {
		return false
	}
	return n.driver.VerifyConnectivity(ctx) == nil
}
