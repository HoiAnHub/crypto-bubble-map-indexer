package blockchain

import (
	"context"
	"errors"
	"strings"
)

// EthereumClient provides blockchain interaction capabilities
type EthereumClient struct {
	rpcURL string
	// In production, this would use actual RPC client like go-ethereum
}

// NewEthereumClient creates a new Ethereum client
func NewEthereumClient(rpcURL string) *EthereumClient {
	return &EthereumClient{
		rpcURL: rpcURL,
	}
}

// GetCode returns the bytecode of a contract address
// Returns empty string for EOA, non-empty for contracts
func (ec *EthereumClient) GetCode(ctx context.Context, address string) (string, error) {
	// Placeholder implementation
	// In production, this would make actual RPC call to eth_getCode

	if !isValidEthereumAddress(address) {
		return "", errors.New("invalid Ethereum address")
	}

	// For now, return empty string (indicating EOA)
	// This would be replaced with actual RPC call:
	// client.CodeAt(ctx, common.HexToAddress(address), nil)
	return "", nil
}

// IsContract checks if address is a contract (has bytecode)
func (ec *EthereumClient) IsContract(ctx context.Context, address string) (bool, error) {
	code, err := ec.GetCode(ctx, address)
	if err != nil {
		return false, err
	}

	// If bytecode exists and is not empty, it's a contract
	return len(code) > 0 && code != "0x", nil
}

// isValidEthereumAddress checks if the address format is valid
func isValidEthereumAddress(address string) bool {
	// Basic validation: starts with 0x and has 42 chars total
	address = strings.ToLower(address)
	if len(address) != 42 {
		return false
	}

	if !strings.HasPrefix(address, "0x") {
		return false
	}

	// Check if remaining chars are valid hex
	hexPart := address[2:]
	for _, char := range hexPart {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}

	return true
}

// MockEthereumClient provides a mock implementation for testing
type MockEthereumClient struct {
	contractAddresses map[string]bool
}

// NewMockEthereumClient creates a mock client with predefined contract addresses
func NewMockEthereumClient() *MockEthereumClient {
	return &MockEthereumClient{
		contractAddresses: make(map[string]bool),
	}
}

// SetContractAddress marks an address as a contract for testing
func (mc *MockEthereumClient) SetContractAddress(address string, isContract bool) {
	mc.contractAddresses[strings.ToLower(address)] = isContract
}

// GetCode mock implementation
func (mc *MockEthereumClient) GetCode(ctx context.Context, address string) (string, error) {
	if !isValidEthereumAddress(address) {
		return "", errors.New("invalid Ethereum address")
	}

	if mc.contractAddresses[strings.ToLower(address)] {
		return "0x608060405234801561001057600080fd5b50", nil // Mock bytecode
	}

	return "", nil // EOA
}

// IsContract mock implementation
func (mc *MockEthereumClient) IsContract(ctx context.Context, address string) (bool, error) {
	code, err := mc.GetCode(ctx, address)
	if err != nil {
		return false, err
	}

	return len(code) > 0, nil
}
