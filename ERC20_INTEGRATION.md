# ERC20 Integration

Dự án crypto-bubble-map-indexer đã được mở rộng để hỗ trợ decode và index các ERC20 Token transfers, tạo ra relationship graphs giữa các wallets thông qua ERC20 token transfers.

## Tính năng mới

### 1. ERC20 Transfer Decoding
- **Decode ERC20 Transfer events** từ transaction data
- Hỗ trợ các ERC20 `transfer(address,uint256)` function calls
- Parse transaction data để extract thông tin transfer: from, to, value, contract address

### 2. ERC20 Contract Management
- **Tự động tạo ERC20Contract nodes** trong Neo4J khi detect ERC20 transfers
- Lưu trữ thông tin contract: address, name, symbol, decimals, network
- Track transaction volume và lần đầu/cuối cùng thấy contract

### 3. ERC20 Transfer Relationships
- **Tạo ERC20_TRANSFER relationships** giữa các Wallet nodes
- Mỗi relationship chứa thông tin: value, tx_hash, timestamp, contract_address
- Tạo INTERACTED_WITH relationships giữa wallets và contracts

## Cấu trúc Database

### Neo4J Graph Structure
```
(Wallet:Address1)-[:ERC20_TRANSFER {value, tx_hash, timestamp, contract_address}]->(Wallet:Address2)
(Wallet:Address1)-[:INTERACTED_WITH]->(ERC20Contract:ContractAddress)
(Wallet:Address2)-[:INTERACTED_WITH]->(ERC20Contract:ContractAddress)
```

### Entities mới
1. **ERC20Transfer**: Represents an ERC20 transfer event
2. **ERC20Contract**: Represents an ERC20 token contract
3. **ERC20TransferRelationship**: Relationship between wallets via token transfer

## Workflow

### Transaction Processing Flow
1. **Regular Transaction Processing**: Vẫn tạo SENT_TO relationships giữa wallets
2. **ERC20 Detection**: Kiểm tra transaction data để tìm ERC20 transfers
3. **Contract Registration**: Tạo/update ERC20Contract node nếu chưa exist
4. **Transfer Relationships**: Tạo ERC20_TRANSFER relationships giữa wallets
5. **Interaction Tracking**: Tạo INTERACTED_WITH relationships

### ERC20 Decoder Service
- **Signature Detection**: Detect `transfer(address,uint256)` method calls (0xa9059cbb)
- **Parameter Parsing**: Extract destination address và transfer amount
- **Data Validation**: Validate transaction data format và length

## API Extensions

### Indexing Service Methods
```go
// Retrieve ERC20 transfers for a specific wallet
GetERC20TransfersForWallet(ctx context.Context, address string, limit int) ([]*entity.ERC20Transfer, error)

// Retrieve ERC20 transfers between two wallets
GetERC20TransfersBetweenWallets(ctx context.Context, fromAddress, toAddress string, limit int) ([]*entity.ERC20Transfer, error)
```

## Neo4J Queries

### Tìm tất cả ERC20 transfers của một wallet
```cypher
MATCH (w:Wallet {address: "0x123..."})-[r:ERC20_TRANSFER]-(other:Wallet)
RETURN r.contract_address, r.value, r.tx_hash, startNode(r).address, endNode(r).address
```

### Tìm ERC20 transfers giữa hai wallets
```cypher
MATCH (from:Wallet {address: $from_address})-[r:ERC20_TRANSFER]->(to:Wallet {address: $to_address})
RETURN r.contract_address, r.value, r.tx_hash, r.timestamp, r.network,
       from.address, to.address
ORDER BY r.timestamp DESC
LIMIT $limit
```

### Tìm các wallets có tương tác với một ERC20 contract
```cypher
MATCH (w:Wallet)-[:INTERACTED_WITH]->(c:ERC20Contract {address: "0xA0b86a33E6..."})
RETURN w.address, w.total_transactions
```

## Components Architecture

### Domain Layer
- `entity/erc20.go`: ERC20 entities (Transfer, Contract, TransferRelationship)
- `service/erc20_decoder.go`: ERC20 decoder service interface
- `repository/erc20_repository.go`: ERC20 repository interface

### Infrastructure Layer
- `blockchain/erc20_decoder.go`: ERC20 decoder implementation
- `database/neo4j_erc20_repository.go`: Neo4J ERC20 repository implementation

### Application Layer
- Enhanced `IndexingApplicationService` để process ERC20 transfers
- Integration với existing transaction processing workflow

## Error Handling

- **Non-blocking**: ERC20 processing errors không làm fail toàn bộ transaction processing
- **Logging**: Chi tiết log cho debugging ERC20 decode issues
- **Graceful degradation**: Nếu không decode được ERC20, vẫn process normal transaction

## Lưu ý Implementation

### Current Limitations
1. **Transaction Data Only**: Hiện tại chỉ decode từ transaction input data, chưa parse transaction logs
2. **Basic ERC20 Detection**: Chỉ detect `transfer(address,uint256)` calls
3. **Contract Info**: Placeholder contract info (name, symbol, decimals)

### Future Enhancements
1. **Transaction Receipt Parsing**: Parse logs để detect Transfer events chính xác hơn
2. **Multiple Transfer Events**: Support multiple transfers trong một transaction
3. **Contract Metadata**: Call contract methods để get real name, symbol, decimals
4. **Transfer Types**: Support transferFrom, approve, và các ERC20 methods khác

## Cách sử dụng

ERC20 functionality sẽ tự động hoạt động khi indexer process transactions. Không cần configuration thêm.

Để query ERC20 data:
1. Sử dụng các API methods mới trong IndexingService
2. Query trực tiếp Neo4J với các queries ở trên
3. Kiểm tra ERC20_TRANSFER relationships trong graph visualization

## Dependencies

- `github.com/ethereum/go-ethereum`: Ethereum client libraries cho ABI parsing và crypto functions
- Existing Neo4J, NATS, và logging infrastructure