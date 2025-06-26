# Enhanced Node Classification với Bytecode Analysis

## Tổng quan

Hệ thống phân loại node đã được cải tiến để phân biệt chính xác giữa địa chỉ hợp đồng (smart contracts) và địa chỉ ví người dùng (EOA - Externally Owned Accounts) bằng cách kiểm tra bytecode tại địa chỉ đó.

## Nguyên lý hoạt động

### 1. Kiểm tra Bytecode
```go
// Nếu bytecode khác rỗng ⇒ contract, ngược lại ⇒ EOA
code, err := blockchain.GetCodeAt(ctx, address, nil)
if len(code) == 0 {
    // EOA (Externally Owned Account)
    return false, entity.NodeTypeUnknown, nil
} else {
    // Smart Contract
    return true, contractType, nil
}
```

### 2. Phân tích Contract Type từ Bytecode
Hệ thống tìm kiếm các function signatures trong bytecode để xác định loại contract:

#### ERC20 Token Contracts
- **Signatures**: `transfer(a9059cbb)` + `approve(095ea7b3)`
- **Classification**: `NodeTypeTokenContract`

#### DEX Contracts
- **Signatures**: `swapExactETHForTokens(7ff36ab5)`, `swapExactTokensForETH(18cbafe5)`
- **Classification**: `NodeTypeDEXContract`

#### WETH Contracts
- **Signatures**: `deposit(d0e30db0)` + `withdraw(2e1a7d4d)`
- **Classification**: `NodeTypeTokenContract`

#### Lending Contracts
- **Signatures**: `mint(a6afed95)`, `redeem(852a12e3)`
- **Classification**: `NodeTypeLendingContract`

#### Proxy Contracts
- **Pattern**: Implementation slot bytecode `360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc`
- **Classification**: `NodeTypeProxyContract`

#### Factory Contracts
- **Pattern**: CREATE2 bytecode `5af43d82803e903d91602b57fd5bf3`
- **Classification**: `NodeTypeFactoryContract`

## Thay đổi trong Code

### 1. NodeClassifierService
```go
// Thêm blockchain service dependency
type NodeClassifierService struct {
    // ... existing fields
    blockchain BlockchainService
    logger     *logger.Logger
}

// Constructor mới
func NewNodeClassifierService(blockchain BlockchainService, logger *logger.Logger) *NodeClassifierService
```

### 2. Method mới: checkAddressType
```go
func (ncs *NodeClassifierService) checkAddressType(ctx context.Context, address string) (isContract bool, contractType entity.NodeType, err error)
```

### 3. Method mới: analyzeContractType
```go
func (ncs *NodeClassifierService) analyzeContractType(bytecode []byte, address string) entity.NodeType
```

### 4. BlockchainService Interface
```go
type BlockchainService interface {
    GetCodeAt(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error)
}
```

### 5. EthereumService Implementation
```go
func (s *EthereumService) GetCodeAt(ctx context.Context, address string, blockNumber *big.Int) ([]byte, error)
```

### 6. Detection Method mới
```go
DetectionMethodBytecodeAnalysis DetectionMethod = "BYTECODE_ANALYSIS"
```

## Luồng Classification mới

1. **Blacklist/Sanctions Check** (ưu tiên cao nhất)
2. **Bytecode Analysis** (kiểm tra EOA vs Contract)
   - Nếu là EOA → set `NodeTypeEOA`, confidence 0.9
   - Nếu là Contract → phân tích contract type từ bytecode
3. **Known Contracts Check**
4. **Exchange Patterns** (phân biệt exchange EOA vs contract)
5. **Transaction Pattern Analysis**
6. **Classification Rules**
7. **Default Classification**
   - Contract: `NodeTypeTokenContract` (most common)
   - EOA: `NodeTypeEOA`

## Cải tiến so với trước

### Trước
- Chỉ dựa vào patterns, rules, known lists
- Không phân biệt được Contract vs EOA
- Nhiều contract bị classify sai thành wallet

### Sau
- ✅ Kiểm tra bytecode để phân biệt chính xác Contract vs EOA
- ✅ Phân tích function signatures để xác định contract type
- ✅ Higher confidence scores (0.9 vs 0.3)
- ✅ Thêm detection method `BYTECODE_ANALYSIS`
- ✅ Tags: `smart_contract` hoặc `eoa`
- ✅ Exchange classification chính xác hơn

## Testing

Chạy test script để kiểm tra:
```bash
cd crypto-bubble-map-indexer
go run scripts/test_node_classification.go
```

### Test Cases
1. **Uniswap Token**: `0x1f9840a85d5af5bf1d1762f925bdaddc4201f984` → `TokenContract`
2. **Uniswap V2 Router**: `0x7a250d5630b4cf539739df2c5dacb4c659f2488d` → `DEXContract`
3. **WETH**: `0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2` → `TokenContract`
4. **Dead Address**: `0x000000000000000000000000000000000000dead` → `EOA`
5. **Regular Wallet**: `0x1234567890123456789012345678901234567890` → `EOA`

## Performance Considerations

1. **Caching**: Nên cache kết quả bytecode để tránh query lặp lại
2. **Rate Limiting**: GetCodeAt calls cần rate limiting
3. **Batch Processing**: Có thể batch multiple addresses
4. **Fallback**: Nếu bytecode check fail, vẫn dùng các methods khác

## Future Improvements

1. **ABI Analysis**: Phân tích ABI để classification chính xác hơn
2. **Event Analysis**: Kiểm tra events để xác định contract behavior
3. **Solidity Version Detection**: Detect compiler version từ bytecode
4. **Dynamic Analysis**: Monitor contract interactions để improve classification
5. **ML-based Classification**: Sử dụng ML models để classify từ bytecode patterns

## Error Handling

- Nếu `GetCodeAt` fail → continue với other classification methods
- Log warnings cho failed bytecode checks
- Maintain backward compatibility với existing classification logic

## Dependencies

- `github.com/ethereum/go-ethereum/ethclient`: Ethereum client
- `github.com/ethereum/go-ethereum/common`: Address handling
- Logger component cho debugging
- Context cho timeout handling