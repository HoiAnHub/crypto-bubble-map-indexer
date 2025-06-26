package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	app_service "crypto-bubble-map-indexer/internal/application/service"
	"crypto-bubble-map-indexer/internal/domain/entity"
	domain_service "crypto-bubble-map-indexer/internal/domain/service"
	"crypto-bubble-map-indexer/internal/infrastructure/blockchain"
	"crypto-bubble-map-indexer/internal/infrastructure/config"
	"crypto-bubble-map-indexer/internal/infrastructure/database"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"
	"crypto-bubble-map-indexer/internal/infrastructure/messaging"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create logger
	log, err := logger.NewLogger(cfg.App.LogLevel)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Create FX application
	app := fx.New(
		// Provide dependencies
		fx.Supply(cfg),
		fx.Supply(log),
		fx.Supply(&cfg.NATS),
		fx.Supply(&cfg.Neo4J),
		fx.Provide(func() *zap.Logger { return log.Logger }),

		// Infrastructure providers
		fx.Provide(
			database.NewNeo4JClient,
			database.NewNeo4JWalletRepository,
			database.NewNeo4JTransactionRepository,
			database.NewNeo4JERC20Repository,
			database.NewNeo4jNodeClassificationRepository,
			blockchain.NewERC20DecoderService,
			func(cfg *config.Config) *blockchain.EthereumClient {
				if cfg.Ethereum.Enabled && cfg.Ethereum.RPCURL != "" {
					return blockchain.NewEthereumClient(cfg.Ethereum.RPCURL)
				}
				// For development, use default client without real RPC
				return blockchain.NewEthereumClient("")
			},
			messaging.NewNATSConsumer,
		),

		// Domain services
		fx.Provide(
			domain_service.NewNodeClassifierService,
		),

		// Application providers
		fx.Provide(
			app_service.NewIndexingApplicationService,
			app_service.NewNodeClassificationAppService,
		),

		// Lifecycle hooks
		fx.Invoke(startIndexer),
		fx.Invoke(startHealthServer),

		// Configure logging
		fx.WithLogger(func() fxevent.Logger {
			return fxevent.NopLogger
		}),
	)

	// Start the application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Error("Failed to start application", zap.Error(err))
		os.Exit(1)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down application...")

	// Stop the application
	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {
		log.Error("Failed to stop application gracefully", zap.Error(err))
		os.Exit(1)
	}

	log.Info("Application stopped successfully")
}

// startIndexer starts the indexing service
func startIndexer(
	lifecycle fx.Lifecycle,
	consumer *messaging.NATSConsumer,
	indexingService domain_service.IndexingService,
	nodeClassifierService *domain_service.NodeClassifierService,
	ethClient *blockchain.EthereumClient,
	log *zap.Logger,
	cfg *config.Config,
	neo4jClient *database.Neo4JClient,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("Starting indexing service...")

			// Connect to Neo4J first
			log.Info("Connecting to Neo4J database")
			if err := neo4jClient.Connect(ctx); err != nil {
				return fmt.Errorf("failed to connect to Neo4J: %w", err)
			}
			log.Info("Successfully connected to Neo4J database")

			// Setup blockchain client for contract detection
			log.Info("Setting up blockchain client for contract detection")
			nodeClassifierService.SetBlockchainClient(ethClient)
			log.Info("Blockchain client configured successfully")

			// Debug: Log NATS configuration
			log.Info("NATS Configuration",
				zap.String("url", cfg.NATS.URL),
				zap.String("stream_name", cfg.NATS.StreamName),
				zap.String("subject_prefix", cfg.NATS.SubjectPrefix),
				zap.Bool("enabled", cfg.NATS.Enabled),
			)

			// Connect to NATS
			if err := consumer.Connect(ctx); err != nil {
				return fmt.Errorf("failed to connect to NATS: %w", err)
			}

			// Start message processing
			go processMessages(ctx, consumer, indexingService, log, cfg)

			log.Info("Indexing service started successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Stopping indexing service...")
			// Close Neo4J connection
			if err := neo4jClient.Close(ctx); err != nil {
				log.Error("Failed to close Neo4J connection", zap.Error(err))
			}
			// Disconnect from NATS
			return consumer.Disconnect()
		},
	})
}

// startHealthServer starts the health check server
func startHealthServer(
	lifecycle fx.Lifecycle,
	cfg *config.Config,
	logger *logger.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting health server...", zap.Int("port", cfg.App.HTTPPort))

			// Create health check server
			mux := http.NewServeMux()
			mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok"}`))
			})

			server := &http.Server{
				Addr:    fmt.Sprintf(":%d", cfg.App.HTTPPort),
				Handler: mux,
			}

			// Start server in background
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("Health server error", zap.Error(err))
				}
			}()

			logger.Info("Health server started successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping health server...")
			return nil
		},
	})
}

// processMessages processes messages from NATS
func processMessages(
	ctx context.Context,
	consumer *messaging.NATSConsumer,
	indexingService domain_service.IndexingService,
	logger *zap.Logger,
	cfg *config.Config,
) {
	msgChan := consumer.GetMessageChannel()
	batch := make([]*entity.Transaction, 0, cfg.App.BatchSize)
	ticker := time.NewTicker(5 * time.Second) // Flush batch every 5 seconds
	defer ticker.Stop()

	// Create a worker pool for parallel batch processing
	type batchJob struct {
		transactions []*entity.Transaction
	}
	jobChan := make(chan batchJob, cfg.App.WorkerPoolSize)
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < cfg.App.WorkerPoolSize; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			logger.Info("Starting batch processing worker", zap.Int("worker_id", workerID))

			for job := range jobChan {
				// Process the batch
				if err := indexingService.ProcessTransactionBatch(ctx, job.transactions); err != nil {
					logger.Error("Failed to process transaction batch",
						zap.Error(err),
						zap.Int("worker_id", workerID),
						zap.Int("batch_size", len(job.transactions)))
				} else {
					logger.Info("Successfully processed batch",
						zap.Int("worker_id", workerID),
						zap.Int("batch_size", len(job.transactions)))
				}
			}
		}(i)
	}

	// Process incoming messages
	for {
		select {
		case <-ctx.Done():
			// Process remaining batch
			if len(batch) > 0 {
				// Clone the batch to avoid race conditions
				txBatch := make([]*entity.Transaction, len(batch))
				copy(txBatch, batch)
				jobChan <- batchJob{transactions: txBatch}
			}

			// Close job channel and wait for workers to finish
			close(jobChan)
			wg.Wait()
			return

		case tx := <-msgChan:
			if tx == nil {
				// Channel closed, clean up
				if len(batch) > 0 {
					// Clone the batch to avoid race conditions
					txBatch := make([]*entity.Transaction, len(batch))
					copy(txBatch, batch)
					jobChan <- batchJob{transactions: txBatch}
				}

				// Close job channel and wait for workers to finish
				close(jobChan)
				wg.Wait()
				return
			}

			batch = append(batch, tx)

			// Process batch if it's full
			if len(batch) >= cfg.App.BatchSize {
				// Clone the batch to avoid race conditions
				txBatch := make([]*entity.Transaction, len(batch))
				copy(txBatch, batch)
				jobChan <- batchJob{transactions: txBatch}

				// Reset batch
				batch = batch[:0]
			}

		case <-ticker.C:
			// Flush batch periodically
			if len(batch) > 0 {
				// Clone the batch to avoid race conditions
				txBatch := make([]*entity.Transaction, len(batch))
				copy(txBatch, batch)
				jobChan <- batchJob{transactions: txBatch}

				// Reset batch
				batch = batch[:0]
			}
		}
	}
}
