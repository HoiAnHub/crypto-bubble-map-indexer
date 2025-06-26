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
	erc20Repo       repository.ERC20Repository
	erc20Decoder    service.ERC20DecoderService
	logger          *logger.Logger
}

// NewIndexingApplicationService creates a new indexing application service
func NewIndexingApplicationService(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	erc20Repo repository.ERC20Repository,
	erc20Decoder service.ERC20DecoderService,
	logger *logger.Logger,
) service.IndexingService {
	return &IndexingApplicationService{
		walletRepo:      walletRepo,
		transactionRepo: transactionRepo,
		erc20Repo:       erc20Repo,
		erc20Decoder:    erc20Decoder,
		logger:          logger.WithComponent("indexing-service"),
	}
}

// ProcessTransaction processes a transaction event and indexes it
func (s *IndexingApplicationService) ProcessTransaction(ctx context.Context, tx *entity.Transaction) error {
	s.logger.Info("Processing transaction", zap.String("hash", tx.Hash))

	// Create or update wallets
	if err := s.createOrUpdateWallets(ctx, tx); err != nil {
		return fmt.Errorf("failed to create/update wallets: %w", err)
	}

	// Create direct relationship between wallets
	rel := &entity.TransactionRelationship{
		FromAddress: tx.From,
		ToAddress:   tx.To,
		Value:       tx.Value,
		GasPrice:    tx.GasPrice,
		Timestamp:   tx.Timestamp,
		TxHash:      tx.Hash,
	}

	if err := s.transactionRepo.CreateTransactionRelationship(ctx, rel); err != nil {
		return fmt.Errorf("failed to create transaction relationship: %w", err)
	}

	// Process ERC20 transfers if applicable
	if err := s.processERC20Transfers(ctx, tx); err != nil {
		s.logger.Error("Failed to process ERC20 transfers",
			zap.String("tx_hash", tx.Hash),
			zap.Error(err))
		// Don't fail the entire transaction processing for ERC20 errors
	}

	s.logger.Info("Successfully processed transaction", zap.String("hash", tx.Hash))
	return nil
}

// ProcessTransactionBatch processes multiple transactions in batch
func (s *IndexingApplicationService) ProcessTransactionBatch(ctx context.Context, transactions []*entity.Transaction) error {
	s.logger.Info("Processing transaction batch", zap.Int("count", len(transactions)))

	// Prepare batch data
	var relationships []*entity.TransactionRelationship
	var erc20Relationships []*entity.ERC20TransferRelationship
	walletMap := make(map[string]*entity.Wallet)
	contractMap := make(map[string]*entity.ERC20Contract)

	// Detailed transaction analysis for debugging
	transactionsWithData := 0
	transactionsToContracts := 0

	for _, tx := range transactions {
		// Count transactions with data for debugging
		if tx.Data != "" && tx.Data != "0x" {
			transactionsWithData++
		}

		// Count transactions to potential contracts (42 char addresses)
		if len(tx.To) == 42 {
			transactionsToContracts++
		}

		// Prepare wallet data
		s.prepareWalletData(tx, walletMap)

		// Prepare regular transaction relationship data
		rel := &entity.TransactionRelationship{
			FromAddress: tx.From,
			ToAddress:   tx.To,
			Value:       tx.Value,
			GasPrice:    tx.GasPrice,
			Timestamp:   tx.Timestamp,
			TxHash:      tx.Hash,
		}
		relationships = append(relationships, rel)

		// Process ERC20 transfers for this transaction with detailed logging
		s.logger.Debug("Attempting ERC20 decode for transaction",
			zap.String("tx_hash", tx.Hash),
			zap.String("from", tx.From),
			zap.String("to", tx.To),
			zap.String("value", tx.Value),
			zap.String("data", tx.Data),
			zap.Bool("has_data", tx.Data != "" && tx.Data != "0x"),
			zap.Bool("to_contract", len(tx.To) == 42))

		erc20Transfers, err := s.erc20Decoder.DecodeERC20Transfer(ctx, tx)
		if err != nil {
			s.logger.Debug("Failed to decode ERC20 transfers",
				zap.String("tx_hash", tx.Hash),
				zap.String("from", tx.From),
				zap.String("to", tx.To),
				zap.String("data", tx.Data),
				zap.Error(err))
			continue // Skip ERC20 processing for this transaction
		}

		// Log if no ERC20 transfers found
		if len(erc20Transfers) == 0 {
			s.logger.Debug("No ERC20 transfers decoded",
				zap.String("tx_hash", tx.Hash),
				zap.String("from", tx.From),
				zap.String("to", tx.To),
				zap.String("data", tx.Data))
			continue
		}

		// Process each ERC20 transfer
		for _, transfer := range erc20Transfers {
			s.logger.Info("Found ERC20 transfer in batch",
				zap.String("tx_hash", tx.Hash),
				zap.String("from", transfer.From),
				zap.String("to", transfer.To),
				zap.String("contract", transfer.ContractAddress),
				zap.String("value", transfer.Value))

			// Prepare ERC20 contract
			if _, exists := contractMap[transfer.ContractAddress]; !exists {
				contractMap[transfer.ContractAddress] = &entity.ERC20Contract{
					Address:   transfer.ContractAddress,
					Name:      "Unknown Token",
					Symbol:    "UNK",
					Decimals:  18,
					FirstSeen: transfer.Timestamp,
					LastSeen:  transfer.Timestamp,
					TotalTxs:  1,
					Network:   transfer.Network,
				}
			} else {
				contractMap[transfer.ContractAddress].LastSeen = transfer.Timestamp
				contractMap[transfer.ContractAddress].TotalTxs++
			}

			// Prepare ERC20 transfer relationship
			transferRel := &entity.ERC20TransferRelationship{
				FromAddress:     transfer.From,
				ToAddress:       transfer.To,
				ContractAddress: transfer.ContractAddress,
				Value:           transfer.Value,
				TxHash:          transfer.TxHash,
				Timestamp:       transfer.Timestamp,
				Network:         transfer.Network,
			}
			erc20Relationships = append(erc20Relationships, transferRel)
		}
	}

	// Log batch analysis results
	s.logger.Info("Batch transaction analysis",
		zap.Int("total_transactions", len(transactions)),
		zap.Int("transactions_with_data", transactionsWithData),
		zap.Int("transactions_to_contracts", transactionsToContracts),
		zap.Int("erc20_transfers_found", len(erc20Relationships)),
		zap.Int("erc20_contracts_found", len(contractMap)))

	// Batch create/update wallets
	if err := s.batchCreateOrUpdateWallets(ctx, walletMap); err != nil {
		return fmt.Errorf("failed to batch create/update wallets: %w", err)
	}

	// Batch create regular transaction relationships
	if err := s.transactionRepo.BatchCreateRelationships(ctx, relationships); err != nil {
		return fmt.Errorf("failed to batch create relationships: %w", err)
	}

	// Batch create/update ERC20 contracts
	contractsCreated := 0
	for _, contract := range contractMap {
		if err := s.erc20Repo.CreateOrUpdateERC20Contract(ctx, contract); err != nil {
			s.logger.Error("Failed to create/update ERC20 contract in batch",
				zap.String("contract", contract.Address),
				zap.Error(err))
			// Continue processing other contracts
		} else {
			contractsCreated++
		}
	}

	// Batch create ERC20 transfer relationships
	if len(erc20Relationships) > 0 {
		s.logger.Info("Attempting to create ERC20 transfer relationships",
			zap.Int("count", len(erc20Relationships)))

		if err := s.erc20Repo.BatchCreateERC20TransferRelationships(ctx, erc20Relationships); err != nil {
			s.logger.Error("Failed to batch create ERC20 transfer relationships",
				zap.Int("count", len(erc20Relationships)),
				zap.Error(err))
			// Don't return error to avoid failing the entire batch
		} else {
			s.logger.Info("Successfully created ERC20 transfer relationships in batch",
				zap.Int("count", len(erc20Relationships)))
		}
	} else {
		s.logger.Info("No ERC20 transfer relationships to create in this batch")
	}

	s.logger.Info("Successfully processed transaction batch",
		zap.Int("count", len(transactions)),
		zap.Int("erc20_transfers", len(erc20Relationships)),
		zap.Int("erc20_contracts_created", contractsCreated))
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

// GetERC20TransfersForWallet retrieves ERC20 transfers for a wallet
func (s *IndexingApplicationService) GetERC20TransfersForWallet(ctx context.Context, address string, limit int) ([]*entity.ERC20Transfer, error) {
	return s.erc20Repo.GetERC20TransfersForWallet(ctx, address, limit)
}

// GetERC20TransfersBetweenWallets retrieves ERC20 transfers between two wallets
func (s *IndexingApplicationService) GetERC20TransfersBetweenWallets(ctx context.Context, fromAddress, toAddress string, limit int) ([]*entity.ERC20Transfer, error) {
	return s.erc20Repo.GetERC20TransfersBetweenWallets(ctx, fromAddress, toAddress, limit)
}

// processERC20Transfers processes ERC20 transfers from a transaction
func (s *IndexingApplicationService) processERC20Transfers(ctx context.Context, tx *entity.Transaction) error {
	// Decode ERC20 transfers from transaction data
	transfers, err := s.erc20Decoder.DecodeERC20Transfer(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to decode ERC20 transfers: %w", err)
	}

	if len(transfers) == 0 {
		return nil // No ERC20 transfers found
	}

	s.logger.Info("Found ERC20 transfers",
		zap.String("tx_hash", tx.Hash),
		zap.Int("count", len(transfers)))

	// Process each ERC20 transfer
	for _, transfer := range transfers {
		// Create ERC20 contract if needed
		contract := &entity.ERC20Contract{
			Address:   transfer.ContractAddress,
			Name:      "Unknown Token",
			Symbol:    "UNK",
			Decimals:  18,
			FirstSeen: transfer.Timestamp,
			LastSeen:  transfer.Timestamp,
			TotalTxs:  1,
			Network:   transfer.Network,
		}

		if err := s.erc20Repo.CreateOrUpdateERC20Contract(ctx, contract); err != nil {
			s.logger.Error("Failed to create/update ERC20 contract",
				zap.String("contract", transfer.ContractAddress),
				zap.Error(err))
			// Continue processing other transfers
		}

		// Create transfer relationship
		transferRel := &entity.ERC20TransferRelationship{
			FromAddress:     transfer.From,
			ToAddress:       transfer.To,
			ContractAddress: transfer.ContractAddress,
			Value:           transfer.Value,
			TxHash:          transfer.TxHash,
			Timestamp:       transfer.Timestamp,
			Network:         transfer.Network,
		}

		if err := s.erc20Repo.CreateERC20TransferRelationship(ctx, transferRel); err != nil {
			s.logger.Error("Failed to create ERC20 transfer relationship",
				zap.String("from", transfer.From),
				zap.String("to", transfer.To),
				zap.String("contract", transfer.ContractAddress),
				zap.Error(err))
			// Continue processing other transfers
		}
	}

	return nil
}
