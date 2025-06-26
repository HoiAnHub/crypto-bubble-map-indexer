package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/domain/service"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

// ERC20DecoderService implements the ERC20 decoder service
type ERC20DecoderService struct {
	logger *logger.Logger
}

// NewERC20DecoderService creates a new ERC20 decoder service
func NewERC20DecoderService(logger *logger.Logger) service.ERC20DecoderService {
	return &ERC20DecoderService{
		logger: logger.WithComponent("erc20-decoder"),
	}
}

// ERC20 Transfer event signature: Transfer(address,address,uint256)
var transferEventSignature = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

// DecodeERC20Transfer decodes ERC20 Transfer events from transaction data
func (s *ERC20DecoderService) DecodeERC20Transfer(ctx context.Context, tx *entity.Transaction) ([]*entity.ERC20Transfer, error) {
	var transfers []*entity.ERC20Transfer

	s.logger.Info("Attempting to decode ERC20 transfer",
		zap.String("tx_hash", tx.Hash),
		zap.String("to", tx.To),
		zap.String("from", tx.From),
		zap.String("value", tx.Value),
		zap.String("data", tx.Data),
		zap.Int("data_length", len(tx.Data)))

	// Check if transaction has data (contract interaction)
	if tx.Data == "" || tx.Data == "0x" {
		s.logger.Debug("No transaction data found, skipping ERC20 decode",
			zap.String("tx_hash", tx.Hash))
		return transfers, nil
	}

	// Check if transaction is to a contract (non-zero address)
	if tx.To == "" || tx.To == "0x0000000000000000000000000000000000000000" {
		s.logger.Debug("Transaction to zero address, skipping ERC20 decode",
			zap.String("tx_hash", tx.Hash))
		return transfers, nil
	}

	// Try to decode as direct transfer call
	transfer, err := s.decodeDirectTransfer(tx)
	if err != nil {
		s.logger.Debug("Failed to decode as direct transfer",
			zap.String("tx_hash", tx.Hash),
			zap.String("data", tx.Data),
			zap.Error(err))

		// Try alternative decoding methods
		transfer, err = s.decodeAlternativeFormats(tx)
		if err != nil {
			s.logger.Debug("Failed all decoding attempts",
				zap.String("tx_hash", tx.Hash),
				zap.Error(err))
			return transfers, nil
		}
	}

	if transfer != nil {
		s.logger.Info("Successfully decoded ERC20 transfer",
			zap.String("tx_hash", tx.Hash),
			zap.String("contract", transfer.ContractAddress),
			zap.String("from", transfer.From),
			zap.String("to", transfer.To),
			zap.String("value", transfer.Value))
		transfers = append(transfers, transfer)
	}

	return transfers, nil
}

// decodeDirectTransfer decodes direct ERC20 transfer function calls
func (s *ERC20DecoderService) decodeDirectTransfer(tx *entity.Transaction) (*entity.ERC20Transfer, error) {
	data := tx.Data

	s.logger.Debug("Decoding direct transfer",
		zap.String("tx_hash", tx.Hash),
		zap.String("original_data", data),
		zap.Int("data_length", len(data)))

	if len(data) < 10 { // 0x + 4 bytes method signature
		return nil, fmt.Errorf("data too short: %d characters", len(data))
	}

	// Remove 0x prefix if present
	if strings.HasPrefix(data, "0x") {
		data = data[2:]
	}

	s.logger.Debug("Processing data without 0x prefix",
		zap.String("tx_hash", tx.Hash),
		zap.String("processed_data", data),
		zap.Int("processed_length", len(data)))

	// Check if it's a transfer function call
	// transfer(address,uint256) method signature: 0xa9059cbb
	transferMethodSig := "a9059cbb"

	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for method signature: %d characters", len(data))
	}

	methodSig := strings.ToLower(data[:8])
	if methodSig != transferMethodSig {
		s.logger.Debug("Not a transfer function call",
			zap.String("tx_hash", tx.Hash),
			zap.String("found_method_sig", methodSig),
			zap.String("expected_method_sig", transferMethodSig))
		return nil, fmt.Errorf("not a transfer function call: found %s, expected %s", methodSig, transferMethodSig)
	}

	s.logger.Debug("Found transfer method signature",
		zap.String("tx_hash", tx.Hash),
		zap.String("method_sig", methodSig))

	// Decode the function parameters
	// transfer(address to, uint256 value)
	// - to: 32 bytes (address padded)
	// - value: 32 bytes (uint256)

	if len(data) < 8+64+64 { // method sig + to address + value
		return nil, fmt.Errorf("insufficient data for transfer function: %d characters, need %d", len(data), 8+64+64)
	}

	// Extract 'to' address (bytes 8-72, take last 40 characters)
	toHex := strings.ToLower(data[8+24 : 8+64]) // Skip padding, get last 20 bytes
	toAddress := "0x" + toHex

	// Extract value (bytes 72-136)
	valueHex := data[8+64 : 8+128]
	value := new(big.Int)
	_, success := value.SetString(valueHex, 16)
	if !success {
		return nil, fmt.Errorf("failed to parse value hex: %s", valueHex)
	}

	s.logger.Debug("Parsed transfer parameters",
		zap.String("tx_hash", tx.Hash),
		zap.String("to_hex", toHex),
		zap.String("to_address", toAddress),
		zap.String("value_hex", valueHex),
		zap.String("value_decimal", value.String()))

	// Validate addresses
	if len(toHex) != 40 {
		return nil, fmt.Errorf("invalid to address length: %d characters", len(toHex))
	}

	// Create ERC20 transfer
	transfer := &entity.ERC20Transfer{
		ContractAddress: tx.To,
		From:            tx.From,
		To:              toAddress,
		Value:           value.String(),
		TxHash:          tx.Hash,
		BlockNumber:     tx.BlockNumber,
		Timestamp:       tx.Timestamp,
		Network:         tx.Network,
	}

	return transfer, nil
}

// decodeAlternativeFormats tries alternative ways to decode ERC20 transfers
func (s *ERC20DecoderService) decodeAlternativeFormats(tx *entity.Transaction) (*entity.ERC20Transfer, error) {
	data := tx.Data

	// Remove 0x prefix if present
	if strings.HasPrefix(data, "0x") {
		data = data[2:]
	}

	s.logger.Debug("Trying alternative decoding formats",
		zap.String("tx_hash", tx.Hash),
		zap.String("data", data))

	// Try detecting common ERC20 method signatures
	if len(data) >= 8 {
		methodSig := strings.ToLower(data[:8])
		s.logger.Debug("Detected method signature",
			zap.String("tx_hash", tx.Hash),
			zap.String("method_sig", methodSig))

		switch methodSig {
		case "a9059cbb": // transfer(address,uint256)
			return s.decodeTransferMethod(tx, data)
		case "23b872dd": // transferFrom(address,address,uint256)
			return s.decodeTransferFromMethod(tx, data)
		default:
			// Log unknown method signatures for debugging
			s.logger.Debug("Unknown method signature",
				zap.String("tx_hash", tx.Hash),
				zap.String("method_sig", methodSig))
		}
	}

	// If we have transaction value > 0 and it's sent to a contract,
	// we might want to create a basic transfer record for analysis
	if tx.Value != "0" && len(tx.To) == 42 {
		s.logger.Debug("Creating basic transfer record for contract interaction with value",
			zap.String("tx_hash", tx.Hash),
			zap.String("to", tx.To),
			zap.String("value", tx.Value))

		// This might be an ETH transfer to a contract that also handles tokens
		// For debugging purposes, we can track this
		transfer := &entity.ERC20Transfer{
			ContractAddress: tx.To,
			From:            tx.From,
			To:              tx.To,
			Value:           tx.Value,
			TxHash:          tx.Hash,
			BlockNumber:     tx.BlockNumber,
			Timestamp:       tx.Timestamp,
			Network:         tx.Network,
		}

		s.logger.Info("Created fallback transfer record",
			zap.String("tx_hash", tx.Hash),
			zap.String("contract", transfer.ContractAddress),
			zap.String("from", transfer.From),
			zap.String("to", transfer.To),
			zap.String("value", transfer.Value))

		return transfer, nil
	}

	return nil, fmt.Errorf("no alternative decoding method succeeded")
}

// decodeTransferMethod decodes transfer(address,uint256) calls
func (s *ERC20DecoderService) decodeTransferMethod(tx *entity.Transaction, data string) (*entity.ERC20Transfer, error) {
	// This is essentially the same as decodeDirectTransfer but separated for clarity
	return s.decodeDirectTransfer(tx)
}

// decodeTransferFromMethod decodes transferFrom(address,address,uint256) calls
func (s *ERC20DecoderService) decodeTransferFromMethod(tx *entity.Transaction, data string) (*entity.ERC20Transfer, error) {
	s.logger.Debug("Decoding transferFrom method",
		zap.String("tx_hash", tx.Hash))

	// transferFrom(address from, address to, uint256 value)
	// - from: 32 bytes (address padded)
	// - to: 32 bytes (address padded)
	// - value: 32 bytes (uint256)

	if len(data) < 8+64+64+64 { // method sig + from + to + value
		return nil, fmt.Errorf("insufficient data for transferFrom function: %d characters", len(data))
	}

	// Extract 'from' address (bytes 8-72, take last 40 characters)
	fromHex := strings.ToLower(data[8+24 : 8+64])
	fromAddress := "0x" + fromHex

	// Extract 'to' address (bytes 72-136, take last 40 characters)
	toHex := strings.ToLower(data[8+64+24 : 8+64+64])
	toAddress := "0x" + toHex

	// Extract value (bytes 136-200)
	valueHex := data[8+64+64 : 8+64+64+64]
	value := new(big.Int)
	_, success := value.SetString(valueHex, 16)
	if !success {
		return nil, fmt.Errorf("failed to parse value hex: %s", valueHex)
	}

	s.logger.Debug("Parsed transferFrom parameters",
		zap.String("tx_hash", tx.Hash),
		zap.String("from_address", fromAddress),
		zap.String("to_address", toAddress),
		zap.String("value_decimal", value.String()))

	// Create ERC20 transfer
	transfer := &entity.ERC20Transfer{
		ContractAddress: tx.To,
		From:            fromAddress,
		To:              toAddress,
		Value:           value.String(),
		TxHash:          tx.Hash,
		BlockNumber:     tx.BlockNumber,
		Timestamp:       tx.Timestamp,
		Network:         tx.Network,
	}

	return transfer, nil
}

// IsERC20Contract checks if an address is an ERC20 contract
func (s *ERC20DecoderService) IsERC20Contract(ctx context.Context, address string) (bool, error) {
	// Simplified check - in a real implementation, you would:
	// 1. Call the contract to check if it has ERC20 methods (balanceOf, transfer, etc.)
	// 2. Check if it emits Transfer events
	// 3. Maintain a cache/database of known ERC20 contracts

	// For now, we'll assume any contract address with sufficient transaction volume could be ERC20
	// This is a placeholder implementation
	return true, nil
}

// GetERC20ContractInfo retrieves ERC20 contract information
func (s *ERC20DecoderService) GetERC20ContractInfo(ctx context.Context, address string) (*entity.ERC20Contract, error) {
	// In a real implementation, you would:
	// 1. Call the contract's name(), symbol(), decimals() methods
	// 2. Cache the results
	// 3. Return the contract info

	// Placeholder implementation
	contract := &entity.ERC20Contract{
		Address:   address,
		Name:      "Unknown Token",
		Symbol:    "UNK",
		Decimals:  18,
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
		TotalTxs:  1,
		Network:   "ethereum",
	}

	return contract, nil
}

// getERC20ABI returns the ERC20 ABI for decoding
func getERC20ABI() abi.ABI {
	// Standard ERC20 ABI
	erc20ABI := `[
		{
			"constant": false,
			"inputs": [
				{"name": "_to", "type": "address"},
				{"name": "_value", "type": "uint256"}
			],
			"name": "transfer",
			"outputs": [{"name": "", "type": "bool"}],
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{"name": "_from", "type": "address"},
				{"name": "_to", "type": "address"},
				{"name": "_value", "type": "uint256"}
			],
			"name": "transferFrom",
			"outputs": [{"name": "", "type": "bool"}],
			"type": "function"
		}
	]`

	erc20ABI_parsed, _ := abi.JSON(strings.NewReader(erc20ABI))
	return erc20ABI_parsed
}
