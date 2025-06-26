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

	// Check if transaction has data (contract interaction)
	if tx.Data == "" || tx.Data == "0x" {
		return transfers, nil
	}

	// Check if transaction is to a contract (non-zero address)
	if tx.To == "" || tx.To == "0x0000000000000000000000000000000000000000" {
		return transfers, nil
	}

	// For now, we'll decode from transaction data if it's a simple ERC20 transfer
	// In a real implementation, you would need to parse transaction receipt logs
	// This is a simplified version that handles direct ERC20 transfer calls

	transfer, err := s.decodeDirectTransfer(tx)
	if err != nil {
		s.logger.Debug("Failed to decode direct transfer",
			zap.String("tx_hash", tx.Hash),
			zap.Error(err))
		return transfers, nil
	}

	if transfer != nil {
		transfers = append(transfers, transfer)
	}

	return transfers, nil
}

// decodeDirectTransfer decodes direct ERC20 transfer function calls
func (s *ERC20DecoderService) decodeDirectTransfer(tx *entity.Transaction) (*entity.ERC20Transfer, error) {
	data := tx.Data
	if len(data) < 10 { // 0x + 4 bytes method signature
		return nil, fmt.Errorf("data too short")
	}

	// Remove 0x prefix if present
	if strings.HasPrefix(data, "0x") {
		data = data[2:]
	}

	// Check if it's a transfer function call
	// transfer(address,uint256) method signature: 0xa9059cbb
	transferMethodSig := "a9059cbb"

	if len(data) < 8 || !strings.HasPrefix(data, transferMethodSig) {
		return nil, fmt.Errorf("not a transfer function call")
	}

	// Decode the function parameters
	// transfer(address to, uint256 value)
	// - to: 32 bytes (address padded)
	// - value: 32 bytes (uint256)

	if len(data) < 8+64+64 { // method sig + to address + value
		return nil, fmt.Errorf("insufficient data for transfer function")
	}

	// Extract 'to' address (bytes 8-72, take last 40 characters)
	toHex := data[8+24 : 8+64] // Skip padding, get last 20 bytes
	toAddress := "0x" + toHex

	// Extract value (bytes 72-136)
	valueHex := data[8+64 : 8+128]
	value := new(big.Int)
	value.SetString(valueHex, 16)

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
		FirstSeen: ctx.Value("timestamp").(time.Time),
		LastSeen:  ctx.Value("timestamp").(time.Time),
		TotalTxs:  0,
		Network:   "ethereum",
	}

	return contract, nil
}

// Helper function to parse ERC20 ABI (simplified)
func getERC20ABI() abi.ABI {
	// Simplified ERC20 ABI - just the Transfer event and transfer function
	const erc20ABI = `[
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "from", "type": "address"},
				{"indexed": true, "name": "to", "type": "address"},
				{"indexed": false, "name": "value", "type": "uint256"}
			],
			"name": "Transfer",
			"type": "event"
		},
		{
			"inputs": [
				{"name": "to", "type": "address"},
				{"name": "value", "type": "uint256"}
			],
			"name": "transfer",
			"outputs": [{"name": "", "type": "bool"}],
			"type": "function"
		}
	]`

	parsedABI, _ := abi.JSON(strings.NewReader(erc20ABI))
	return parsedABI
}
