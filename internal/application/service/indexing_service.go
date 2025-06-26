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
	walletRepo         repository.WalletRepository
	transactionRepo    repository.TransactionRepository
	erc20Repo          repository.ERC20Repository
	erc20Decoder       service.ERC20DecoderService
	nodeClassifier     *service.NodeClassifierService
	classificationRepo repository.NodeClassificationRepository
	logger             *logger.Logger
}

// NewIndexingApplicationService creates a new indexing application service
func NewIndexingApplicationService(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	erc20Repo repository.ERC20Repository,
	erc20Decoder service.ERC20DecoderService,
	nodeClassifier *service.NodeClassifierService,
	classificationRepo repository.NodeClassificationRepository,
	logger *logger.Logger,
) service.IndexingService {
	return &IndexingApplicationService{
		walletRepo:         walletRepo,
		transactionRepo:    transactionRepo,
		erc20Repo:          erc20Repo,
		erc20Decoder:       erc20Decoder,
		nodeClassifier:     nodeClassifier,
		classificationRepo: classificationRepo,
		logger:             logger.WithComponent("indexing-service"),
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

		// Process ERC20 transfers for this transaction
		transfers, err := s.erc20Decoder.DecodeERC20Transfer(ctx, tx)
		if err != nil {
			s.logger.Warn("Failed to decode ERC20 transfers",
				zap.String("tx_hash", tx.Hash),
				zap.Error(err))
		} else if len(transfers) > 0 {
			for _, transfer := range transfers {
				// Create relationship for this transfer/interaction
				relationship := &entity.ERC20TransferRelationship{
					FromAddress:      transfer.From,
					ToAddress:        transfer.To,
					ContractAddress:  transfer.ContractAddress,
					Value:            transfer.Value,
					TxHash:           transfer.TxHash,
					Timestamp:        transfer.Timestamp,
					Network:          transfer.Network,
					InteractionType:  transfer.InteractionType,
					TotalValue:       transfer.Value,
					TransactionCount: 1,
					FirstInteraction: transfer.Timestamp,
					LastInteraction:  transfer.Timestamp,
				}
				erc20Relationships = append(erc20Relationships, relationship)

				// Track unique contracts for creation
				if _, exists := contractMap[transfer.ContractAddress]; !exists && transfer.ContractAddress != "ETH" {
					contractMap[transfer.ContractAddress] = s.createEnhancedContract(transfer)
				}

				s.logger.Debug("Processed contract interaction",
					zap.String("tx_hash", tx.Hash),
					zap.String("interaction_type", string(transfer.InteractionType)),
					zap.String("from", transfer.From),
					zap.String("to", transfer.To),
					zap.String("contract", transfer.ContractAddress),
					zap.String("value", transfer.Value))
			}
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
	s.logger.Info("About to create regular transaction relationships",
		zap.Int("relationships_count", len(relationships)))

	if len(relationships) > 0 {
		s.logger.Info("Sample relationships data",
			zap.String("first_from", relationships[0].FromAddress),
			zap.String("first_to", relationships[0].ToAddress),
			zap.String("first_value", relationships[0].Value),
			zap.String("first_hash", relationships[0].TxHash))
	}

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

	// Classify sender wallet
	if err := s.classifyWallet(ctx, tx.From); err != nil {
		s.logger.Warn("Failed to classify sender wallet",
			zap.String("address", tx.From),
			zap.Error(err))
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

	// Classify receiver wallet
	if err := s.classifyWallet(ctx, tx.To); err != nil {
		s.logger.Warn("Failed to classify receiver wallet",
			zap.String("address", tx.To),
			zap.Error(err))
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

		// Classify each wallet after creation/update
		if err := s.classifyWallet(ctx, wallet.Address); err != nil {
			s.logger.Warn("Failed to classify wallet in batch",
				zap.String("address", wallet.Address),
				zap.Error(err))
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

// determineContractType determines the contract type based on interaction type and classifier
func (s *IndexingApplicationService) determineContractType(interactionType entity.ContractInteractionType, contractAddress string, methodSignature string) string {
	// First, try to get a more specific classification if we have a classifier
	// For now, use basic mapping based on interaction type
	switch interactionType {
	case entity.InteractionSwap:
		// Try to detect specific DEX types
		switch methodSignature {
		case "7ff36ab5", "18cbafe5", "38ed1739":
			return "UNISWAP_V2"
		case "022c0d9f":
			return "UNISWAP_V2"
		default:
			return "DEX"
		}
	case entity.InteractionAddLiquidity, entity.InteractionRemoveLiquidity:
		return "LIQUIDITY_POOL"
	case entity.InteractionDeposit, entity.InteractionWithdraw:
		// Try to detect specific lending protocols
		switch methodSignature {
		case "a6afed95", "852a12e3":
			return "COMPOUND"
		case "d65d7f80", "69328dec":
			return "AAVE"
		default:
			return "DEFI_PROTOCOL"
		}
	case entity.InteractionMulticall:
		return "MULTICALL"
	case entity.InteractionTransfer, entity.InteractionTransferFrom,
		entity.InteractionApprove, entity.InteractionIncreaseAllowance, entity.InteractionDecreaseAllowance:
		return "ERC20"
	case entity.InteractionETHTransfer:
		return "ETH"
	default:
		return "UNKNOWN"
	}
}

// createEnhancedContract creates an enhanced contract with better classification
func (s *IndexingApplicationService) createEnhancedContract(transfer *entity.ERC20Transfer) *entity.ERC20Contract {
	contractType := s.determineContractType(transfer.InteractionType, transfer.ContractAddress, transfer.MethodSignature)

	// Determine if contract is verified based on known signatures
	isVerified := s.isKnownContract(transfer.MethodSignature)

	return &entity.ERC20Contract{
		Address:      transfer.ContractAddress,
		Name:         s.generateContractName(contractType, transfer.ContractAddress),
		Symbol:       s.generateContractSymbol(contractType),
		Decimals:     18,
		FirstSeen:    transfer.Timestamp,
		LastSeen:     transfer.Timestamp,
		TotalTxs:     1,
		Network:      transfer.Network,
		ContractType: contractType,
		IsVerified:   isVerified,
	}
}

// isKnownContract checks if this is a known contract based on method signatures
func (s *IndexingApplicationService) isKnownContract(methodSignature string) bool {
	knownSignatures := map[string]bool{
		"7ff36ab5": true, // Uniswap swapExactETHForTokens
		"18cbafe5": true, // Uniswap swapExactTokensForETH
		"38ed1739": true, // Uniswap swapExactTokensForTokens
		"a6afed95": true, // Compound mint
		"852a12e3": true, // Compound redeem
		"d65d7f80": true, // Aave deposit
		"69328dec": true, // Aave withdraw
		"ac9650d8": true, // Multicall
		"d0e30db0": true, // WETH deposit
		"2e1a7d4d": true, // WETH withdraw
	}

	return knownSignatures[methodSignature]
}

// generateContractName generates a descriptive name for the contract
func (s *IndexingApplicationService) generateContractName(contractType, address string) string {
	switch contractType {
	case "UNISWAP_V2":
		return "Uniswap V2 Router"
	case "DEX":
		return "DEX Contract"
	case "COMPOUND":
		return "Compound Protocol"
	case "AAVE":
		return "Aave Protocol"
	case "DEFI_PROTOCOL":
		return "DeFi Protocol"
	case "LIQUIDITY_POOL":
		return "Liquidity Pool"
	case "MULTICALL":
		return "Multicall Contract"
	case "ERC20":
		return "ERC20 Token"
	default:
		return fmt.Sprintf("Contract %s", address[:10]+"...")
	}
}

// generateContractSymbol generates a symbol for the contract
func (s *IndexingApplicationService) generateContractSymbol(contractType string) string {
	switch contractType {
	case "UNISWAP_V2":
		return "UNI-V2"
	case "DEX":
		return "DEX"
	case "COMPOUND":
		return "COMP"
	case "AAVE":
		return "AAVE"
	case "DEFI_PROTOCOL":
		return "DEFI"
	case "LIQUIDITY_POOL":
		return "LP"
	case "MULTICALL":
		return "MULTI"
	case "ERC20":
		return "ERC20"
	default:
		return "UNK"
	}
}

// classifyWallet classifies a wallet address using the node classification service
func (s *IndexingApplicationService) classifyWallet(ctx context.Context, address string) error {
	if s.nodeClassifier == nil || s.classificationRepo == nil {
		return nil // Skip classification if services not available
	}

	// Get wallet stats for better classification
	stats, err := s.walletRepo.GetWalletStats(ctx, address)
	if err != nil {
		s.logger.Debug("Could not get wallet stats for classification",
			zap.String("address", address),
			zap.Error(err))
		stats = nil // Continue without stats
	}

	// Perform classification
	classification, err := s.nodeClassifier.ClassifyNode(ctx, address, stats, []string{})
	if err != nil {
		return fmt.Errorf("failed to classify node %s: %w", address, err)
	}

	// Save classification
	if err := s.classificationRepo.CreateOrUpdateClassification(ctx, classification); err != nil {
		return fmt.Errorf("failed to save classification for %s: %w", address, err)
	}

	s.logger.Debug("Successfully classified wallet",
		zap.String("address", address),
		zap.String("node_type", string(classification.PrimaryType)),
		zap.String("risk_level", string(classification.RiskLevel)),
		zap.Float64("confidence", classification.ConfidenceScore))

	return nil
}
