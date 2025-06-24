package service

import (
	"context"
	"fmt"
	"strconv"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/domain/repository"
	"crypto-bubble-map-indexer/internal/domain/service"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"go.uber.org/zap"
)

// IndexingApplicationService implements IndexingService interface
type IndexingApplicationService struct {
	walletRepo      repository.WalletRepository
	transactionRepo repository.TransactionRepository
	logger          *logger.Logger
}

// NewIndexingApplicationService creates a new indexing application service
func NewIndexingApplicationService(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	logger *logger.Logger,
) service.IndexingService {
	return &IndexingApplicationService{
		walletRepo:      walletRepo,
		transactionRepo: transactionRepo,
		logger:          logger.WithComponent("indexing-service"),
	}
}

// ProcessTransaction processes a transaction event and indexes it
func (s *IndexingApplicationService) ProcessTransaction(ctx context.Context, tx *entity.Transaction) error {
	s.logger.Info("Processing transaction", zap.String("hash", tx.Hash))

	// Create transaction node
	txNode := &entity.TransactionNode{
		Hash:        tx.Hash,
		BlockNumber: tx.BlockNumber,
		Value:       tx.Value,
		GasUsed:     tx.GasUsed,
		GasPrice:    tx.GasPrice,
		Timestamp:   tx.Timestamp,
		Network:     tx.Network,
	}

	if err := s.transactionRepo.CreateTransaction(ctx, txNode); err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Create or update wallets
	if err := s.createOrUpdateWallets(ctx, tx); err != nil {
		return fmt.Errorf("failed to create/update wallets: %w", err)
	}

	// Create transaction relationship
	if err := s.createTransactionRelationship(ctx, tx); err != nil {
		return fmt.Errorf("failed to create transaction relationship: %w", err)
	}

	s.logger.Info("Successfully processed transaction", zap.String("hash", tx.Hash))
	return nil
}

// ProcessTransactionBatch processes multiple transactions in batch
func (s *IndexingApplicationService) ProcessTransactionBatch(ctx context.Context, transactions []*entity.Transaction) error {
	s.logger.Info("Processing transaction batch", zap.Int("count", len(transactions)))

	// Prepare batch data
	var txNodes []*entity.TransactionNode
	var relationships []*entity.TransactionRelationship
	walletMap := make(map[string]*entity.Wallet)

	for _, tx := range transactions {
		// Create transaction node
		txNode := &entity.TransactionNode{
			Hash:        tx.Hash,
			BlockNumber: tx.BlockNumber,
			Value:       tx.Value,
			GasUsed:     tx.GasUsed,
			GasPrice:    tx.GasPrice,
			Timestamp:   tx.Timestamp,
			Network:     tx.Network,
		}
		txNodes = append(txNodes, txNode)

		// Prepare wallet data
		s.prepareWalletData(tx, walletMap)

		// Prepare relationship data
		rel := &entity.TransactionRelationship{
			FromAddress: tx.From,
			ToAddress:   tx.To,
			Value:       tx.Value,
			GasPrice:    tx.GasPrice,
			Timestamp:   tx.Timestamp,
			TxHash:      tx.Hash,
		}
		relationships = append(relationships, rel)
	}

	// Batch create transactions
	if err := s.transactionRepo.BatchCreateTransactions(ctx, txNodes); err != nil {
		return fmt.Errorf("failed to batch create transactions: %w", err)
	}

	// Batch create/update wallets
	if err := s.batchCreateOrUpdateWallets(ctx, walletMap); err != nil {
		return fmt.Errorf("failed to batch create/update wallets: %w", err)
	}

	// Batch create relationships
	if err := s.transactionRepo.BatchCreateRelationships(ctx, relationships); err != nil {
		return fmt.Errorf("failed to batch create relationships: %w", err)
	}

	s.logger.Info("Successfully processed transaction batch", zap.Int("count", len(transactions)))
	return nil
}

// GetWalletAnalytics retrieves analytics for a wallet
func (s *IndexingApplicationService) GetWalletAnalytics(ctx context.Context, address string) (*entity.WalletStats, error) {
	return s.walletRepo.GetWalletStats(ctx, address)
}

// GetBubbleAnalysis retrieves bubble analysis data
func (s *IndexingApplicationService) GetBubbleAnalysis(ctx context.Context, minConnections int, limit int) ([]*entity.Wallet, error) {
	return s.walletRepo.GetBubbleWallets(ctx, minConnections, limit)
}

// GetTransactionPath finds transaction path between wallets
func (s *IndexingApplicationService) GetTransactionPath(ctx context.Context, fromAddress, toAddress string, maxHops int) ([]*entity.TransactionNode, error) {
	return s.transactionRepo.GetTransactionPath(ctx, fromAddress, toAddress, maxHops)
}

// createOrUpdateWallets creates or updates wallet entities
func (s *IndexingApplicationService) createOrUpdateWallets(ctx context.Context, tx *entity.Transaction) error {
	// Create/update sender wallet
	senderWallet := &entity.Wallet{
		Address:           tx.From,
		FirstSeen:         tx.Timestamp,
		LastSeen:          tx.Timestamp,
		TotalTransactions: 1,
		TotalSent:         tx.Value,
		TotalReceived:     "0",
		Network:           tx.Network,
	}

	if err := s.walletRepo.CreateOrUpdateWallet(ctx, senderWallet); err != nil {
		return fmt.Errorf("failed to create/update sender wallet: %w", err)
	}

	// Create/update receiver wallet
	receiverWallet := &entity.Wallet{
		Address:           tx.To,
		FirstSeen:         tx.Timestamp,
		LastSeen:          tx.Timestamp,
		TotalTransactions: 1,
		TotalSent:         "0",
		TotalReceived:     tx.Value,
		Network:           tx.Network,
	}

	if err := s.walletRepo.CreateOrUpdateWallet(ctx, receiverWallet); err != nil {
		return fmt.Errorf("failed to create/update receiver wallet: %w", err)
	}

	return nil
}

// createTransactionRelationship creates the relationship between wallets
func (s *IndexingApplicationService) createTransactionRelationship(ctx context.Context, tx *entity.Transaction) error {
	rel := &entity.TransactionRelationship{
		FromAddress: tx.From,
		ToAddress:   tx.To,
		Value:       tx.Value,
		GasPrice:    tx.GasPrice,
		Timestamp:   tx.Timestamp,
		TxHash:      tx.Hash,
	}

	return s.transactionRepo.CreateTransactionRelationship(ctx, rel)
}

// prepareWalletData prepares wallet data for batch processing
func (s *IndexingApplicationService) prepareWalletData(tx *entity.Transaction, walletMap map[string]*entity.Wallet) {
	// Prepare sender wallet
	if wallet, exists := walletMap[tx.From]; exists {
		wallet.LastSeen = tx.Timestamp
		wallet.TotalTransactions++
		value, _ := strconv.ParseFloat(wallet.TotalSent, 64)
		txValue, _ := strconv.ParseFloat(tx.Value, 64)
		wallet.TotalSent = fmt.Sprintf("%.0f", value+txValue)
	} else {
		walletMap[tx.From] = &entity.Wallet{
			Address:           tx.From,
			FirstSeen:         tx.Timestamp,
			LastSeen:          tx.Timestamp,
			TotalTransactions: 1,
			TotalSent:         tx.Value,
			TotalReceived:     "0",
			Network:           tx.Network,
		}
	}

	// Prepare receiver wallet
	if wallet, exists := walletMap[tx.To]; exists {
		wallet.LastSeen = tx.Timestamp
		wallet.TotalTransactions++
		value, _ := strconv.ParseFloat(wallet.TotalReceived, 64)
		txValue, _ := strconv.ParseFloat(tx.Value, 64)
		wallet.TotalReceived = fmt.Sprintf("%.0f", value+txValue)
	} else {
		walletMap[tx.To] = &entity.Wallet{
			Address:           tx.To,
			FirstSeen:         tx.Timestamp,
			LastSeen:          tx.Timestamp,
			TotalTransactions: 1,
			TotalSent:         "0",
			TotalReceived:     tx.Value,
			Network:           tx.Network,
		}
	}
}

// batchCreateOrUpdateWallets creates or updates wallets in batch
func (s *IndexingApplicationService) batchCreateOrUpdateWallets(ctx context.Context, walletMap map[string]*entity.Wallet) error {
	for _, wallet := range walletMap {
		if err := s.walletRepo.CreateOrUpdateWallet(ctx, wallet); err != nil {
			return fmt.Errorf("failed to create/update wallet %s: %w", wallet.Address, err)
		}
	}
	return nil
}
