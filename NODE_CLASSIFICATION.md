# Node Classification System

## Tổng quan

Hệ thống Node Classification được thiết kế để tự động phân loại và đánh giá rủi ro các địa chỉ blockchain, giúp tăng cường khả năng tracing và phân tích trong hệ thống crypto bubble map indexer.

## Các loại Node được hỗ trợ

### 1. Address Node Types (Loại địa chỉ)

#### Wallet Types
- **EOA (Externally Owned Account)**: Ví của người dùng bình thường
- **Exchange Wallet**: Ví của sàn giao dịch tập trung
- **Exchange Hot Wallet**: Ví nóng của sàn (giao dịch thường xuyên)
- **Exchange Cold Wallet**: Ví lạnh của sàn (lưu trữ dài hạn)
- **Bridge Wallet**: Ví của các bridge cross-chain

#### Bot & Automation Types
- **MEV Bot**: Bot Maximum Extractable Value
- **Arbitrage Bot**: Bot chênh lệch giá
- **Market Maker**: Dịch vụ tạo lập thị trường

#### High-Value Types
- **Whale**: Cá voi (tài khoản có giá trị cao)

#### Risk Types
- **Mixer Wallet**: Địa chỉ liên quan đến rửa tiền/privacy mixer
- **Suspicious Wallet**: Ví bị đánh dấu nghi ngờ
- **Blacklisted Wallet**: Ví bị đưa vào blacklist chính thức

### 2. Contract Node Types (Loại hợp đồng)

#### DeFi Contracts
- **DEX Contract**: Hợp đồng sàn phi tập trung
- **Lending Contract**: Hợp đồng cho vay
- **Staking Contract**: Hợp đồng staking
- **Yield Contract**: Hợp đồng yield farming
- **Insurance Contract**: Hợp đồng bảo hiểm DeFi

#### Infrastructure Contracts
- **Bridge Contract**: Hợp đồng bridge cross-chain
- **Oracle Contract**: Hợp đồng oracle dữ liệu
- **Multisig Contract**: Hợp đồng multi-signature
- **Proxy Contract**: Hợp đồng proxy/upgradeable
- **Factory Contract**: Hợp đồng factory

#### Token Contracts
- **Token Contract**: Hợp đồng token (ERC20/ERC721/ERC1155)

#### High-Risk Contracts
- **Gambling Contract**: Hợp đồng cờ bạc/game
- **Ponzi Contract**: Hợp đồng lừa đảo/Ponzi
- **Privacy Contract**: Hợp đồng bảo mật (như Tornado Cash)

### 3. Service Node Types (Loại dịch vụ)

- **Oracle Service**: Dịch vụ cung cấp dữ liệu oracle
- **Flash Loan Provider**: Nhà cung cấp vay nhanh
- **Yield Aggregator**: Bộ tổng hợp yield farming
- **Liquidity Provider**: Nhà cung cấp thanh khoản
- **Validator**: Validator của PoS
- **Miner**: Thợ đào PoW

### 4. Exchange-Specific Types

- **CEX Deposit**: Địa chỉ nạp tiền sàn tập trung
- **CEX Withdrawal**: Địa chỉ rút tiền sàn tập trung
- **CEX Settlement**: Địa chỉ thanh toán sàn tập trung

### 5. Special Categories (Loại đặc biệt)

- **Dark Web**: Liên quan đến dark web
- **Ransomware**: Liên quan đến ransomware
- **Terrorist Financing**: Tài trợ khủng bố
- **Money Laundering**: Rửa tiền
- **Sanctioned**: Bị trừng phạt chính phủ

## Cấp độ rủi ro

### Risk Levels
- **LOW**: Rủi ro thấp (EOA, Exchange wallet hợp pháp)
- **MEDIUM**: Rủi ro trung bình (MEV bot, Privacy contract)
- **HIGH**: Rủi ro cao (Mixer wallet, Gambling contract)
- **CRITICAL**: Rủi ro nghiêm trọng (Blacklisted, Sanctioned, Ransomware)

## Kiến trúc hệ thống

### Core Components

1. **NodeClassifierService**: Service chính để phân loại node
2. **NodeClassificationRepository**: Repository để lưu trữ thông tin phân loại
3. **NodeClassificationAppService**: Application service tích hợp vào hệ thống

### Entities

1. **NodeClassification**: Entity chính lưu thông tin phân loại
2. **NodeRelationship**: Mối quan hệ giữa các node
3. **ClassificationRule**: Quy tắc phân loại

## Cách sử dụng

### 1. Phân loại một địa chỉ

```go
// Khởi tạo service
classifier := service.NewNodeClassifierService()
appService := service.NewNodeClassificationAppService(
    classifier, walletRepo, classificationRepo, transactionRepo)

// Phân loại địa chỉ
classification, err := appService.ClassifyWalletAddress(ctx, "0x1234...")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Address: %s\n", classification.Address)
fmt.Printf("Type: %s\n", classification.PrimaryType)
fmt.Printf("Risk Level: %s\n", classification.RiskLevel)
fmt.Printf("Confidence: %.2f\n", classification.ConfidenceScore)
```

### 2. Phân loại hàng loạt

```go
addresses := []string{"0x1234...", "0x5678...", "0x9abc..."}
classifications, err := appService.BulkClassifyAddresses(ctx, addresses)
```

### 3. Tìm kiếm địa chỉ nghi ngờ

```go
suspiciousAddresses, err := appService.GetSuspiciousAddresses(ctx)
```

### 4. Phân tích cluster nghi ngờ

```go
cluster, err := appService.AnalyzeSuspiciousCluster(ctx, "0x1234...", 3)
```

### 5. Cập nhật blacklist

```go
err := appService.UpdateBlacklist(ctx, "0x1234...", "Known ransomware address")
```

## Quy tắc phân loại

### Pattern-based Classification

Hệ thống sử dụng nhiều phương pháp để phân loại:

1. **Pattern Analysis**: Phân tích mẫu giao dịch
2. **Known Address Database**: Cơ sở dữ liệu địa chỉ đã biết (200+ exchanges, 80+ DeFi, 100+ risk addresses)
3. **Behavioral Analysis**: Phân tích hành vi
4. **ML Models**: Mô hình machine learning (tương lai)
5. **Manual Classification**: Phân loại thủ công

## Enhanced Address Database

### Exchange Address Patterns (200+ Addresses)

**Global Major Exchanges:**
- **Binance**: 10+ hot wallets, cold storage addresses
- **Coinbase**: 10+ Pro and retail wallets
- **Kraken**: 8+ exchange wallets with different tiers
- **Huobi**: 7+ exchange addresses including cold storage
- **OKEx**: 5+ trading and deposit wallets
- **Bitfinex**: 5+ exchange addresses
- **Gemini**: 4+ institutional-grade wallets

**Regional & Specialized Exchanges:**
- **Asian Markets**: Bithumb, Upbit, Coinone, Korbit (Korea), BitFlyer, Coincheck, Zaif (Japan)
- **Indian Subcontinent**: WazirX, CoinDCX
- **European**: Bitstamp, Luno, BTCTurk
- **North American**: FTX, Crypto.com, KuCoin, Gate.io, Bittrex, Poloniex
- **Canadian**: Coinsquare, Bitbuy

### DeFi Protocol Addresses (80+ Contracts)

**Decentralized Exchanges:**
- **Uniswap Ecosystem**: V2/V3 Routers, Factory contracts, UNI token
- **SushiSwap**: Router, Factory, SUSHI token
- **Curve Finance**: 3Pool, Compound Pool, sETH Pool, CRV token
- **Balancer**: V2 Vault, BAL token
- **1inch**: V3/V4 Aggregator Routers, 1INCH token

**Lending & Borrowing:**
- **Compound**: Comptroller, cTokens (cDAI, cUSDC, cETH, cUSDT), COMP governance
- **Aave**: Lending Pool V2, ETH Gateway, AAVE token, Provider contracts
- **MakerDAO**: Vat core, PSM stability module, MKR governance, DAI stablecoin

**Infrastructure & Oracles:**
- **Chainlink**: LINK token, Price Feeds (ETH/USD, BTC/USD, USDC/USD, nhiều pairs khác)
- **Yearn Finance**: YFI token, Yield Vaults (yUSDC, yETH, yDAI, yBTC)

**Token Standards:**
- **Wrapped Assets**: WETH, WBTC
- **Stablecoins**: USDC, USDT, BUSD, PAXG
- **Major Tokens**: REP, ZRX, BAT, OMG, SNT
- **Gaming/Metaverse**: MANA (Decentraland), SAND (Sandbox), ENJ (Enjin)

**Cross-chain & Privacy:**
- **Bridge Protocols**: Wormhole, Hop Protocol, Arbitrum Bridge, Optimism Gateway, Polygon Bridge
- **Privacy Contracts**: Tornado Cash (multiple denominations: 0.1-100 ETH, 100-100k DAI)
- **NFT Infrastructure**: OpenSea, LooksRare, Foundation, X2Y2
- **Multisig Solutions**: Gnosis Safe, known celebrity/influencer multisigs

### Risk & Compliance Database (100+ Addresses)

**Blacklisted Addresses - Major Exploits:**
- **Historic Hacks**: The DAO exploit, Parity wallet hack, Coincheck incident
- **DeFi Exploits**: Cream Finance, Harvest Finance, Cover Protocol, Akropolis, Pickle Finance, EasyFi, Rari Capital
- **Bridge Hacks**: Poly Network, Ronin Bridge, Wormhole hack
- **Recent Incidents**: Beanstalk, Terra Luna collapse

**Scam & Fraud Operations:**
- **Ponzi Schemes**: MMM Global, PlusToken, OneCoin, BitConnect, Cloud Token
- **Fake Token Contracts**: Scam clones of EOS, Tether, Bitcoin, Ethereum
- **Rug Pulls**: Various DeFi rug pull addresses

**Ransomware & Cybercrime:**
- **Ransomware Families**: WannaCry, Maze, REvil, Darkside, Conti
- **APT Groups**: Lazarus Group, North Korea state actors, Russian cybercriminals, Iranian hackers, Chinese state actors

**Sanctioned Entities (OFAC/EU/UN Compliance):**
- **OFAC Sanctions**: Lazarus Group wallets, Iranian sanctions, Russian oligarchs
- **Terrorism Financing**: ISIS/ISIL, Al-Qaeda, Taliban-related addresses
- **Proliferation Financing**: North Korea nuclear program funding
- **OFAC Tornado Cash**: All sanctioned mixer contract addresses and deployer
- **Drug & Human Trafficking**: Major darknet market addresses, trafficking organizations

**Additional Risk Categories:**
- **Counter-terrorism**: International watch list addresses
- **Environmental Crime**: Wildlife trafficking, illegal logging
- **Cyber Warfare**: State-sponsored cyber units

### Dynamic Updates & Maintenance

**Real-time Updates:**
- OFAC sanctions list auto-sync
- Exchange address pattern updates
- New DeFi protocol integration
- Blacklist additions from security feeds

**Pattern Management API:**
```go
// Add new exchange
service.AddExchangePattern("new_exchange", []string{"pattern1", "pattern2"})

// Update known contract
service.AddKnownContract("0x123...", entity.NodeTypeDEXContract)

// Bulk blacklist update
service.UpdateBlacklist(map[string]string{
    "0xabc...": "New hack incident",
    "0xdef...": "Confirmed scam",
})
```

### Classification Rules

```go
// Ví dụ quy tắc phân loại MEV Bot
{
    NodeType: entity.NodeTypeMEVBot,
    RequiredPatterns: []string{"flashloan", "arbitrage"},
    OptionalPatterns: []string{"mev", "sandwich", "frontrun"},
    MinTransactions: 1000,
    MinConfidence: 0.7,
    Weight: 1.0,
}
```

## Database Schema

### Neo4j Labels và Properties

```cypher
// Wallet node với classification
(:Wallet {
    address: string,
    node_type: string,
    risk_level: string,
    confidence_score: float,
    secondary_types: string, // JSON array
    detection_methods: string, // JSON array
    tags: string, // JSON array
    exchanges: string, // JSON array
    protocols: string, // JSON array
    last_classified: datetime,
    classification_count: int
})

// Relationships
(:Wallet)-[:FUNDING]->(:Wallet)
(:Wallet)-[:CONTROLLED_BY]->(:Wallet)
(:Wallet)-[:SIMILAR_PATTERN]->(:Wallet)
(:Wallet)-[:SUSPICIOUS]->(:Wallet)
(:Wallet)-[:BLACKLISTED]->(:Blacklist)
```

## API Endpoints (Tương lai)

```
GET /api/v1/classification/{address}
POST /api/v1/classification/bulk
GET /api/v1/classification/suspicious
GET /api/v1/classification/exchange/{exchange}
GET /api/v1/classification/cluster/{address}
POST /api/v1/blacklist
DELETE /api/v1/blacklist/{address}
```

## Monitoring và Alerting

### High-Risk Monitoring

Hệ thống tự động monitor:
- Giao dịch từ/đến địa chỉ high-risk
- Cluster analysis cho các mối quan hệ nghi ngờ
- Pattern detection cho hoạt động bất thường

### Alerts

- **Critical Alert**: Giao dịch với địa chỉ bị sanctioned
- **High Alert**: Giao dịch với mixer wallet
- **Medium Alert**: Unusual pattern detected

## Performance Considerations

### Caching Strategy

- Cache classification results
- Cache known address patterns
- Cache blacklist data

### Batch Processing

- Bulk classification cho performance
- Background re-classification
- Incremental updates

## Security và Compliance

### Data Protection

- Mã hóa dữ liệu nhạy cảm
- Access control cho classification data
- Audit trail cho tất cả thay đổi

### Regulatory Compliance

- OFAC sanctions list integration
- EU sanctions compliance
- Custom sanctions list support

## Tích hợp với hệ thống hiện tại

### Transaction Processing

```go
// Trong transaction processor
func (p *TransactionProcessor) ProcessTransaction(tx *entity.Transaction) error {
    // Process transaction normally
    err := p.processTransaction(tx)
    if err != nil {
        return err
    }

    // Classify addresses if needed
    if !p.isClassified(tx.From) {
        go p.classificationService.ClassifyWalletAddress(context.Background(), tx.From)
    }
    if !p.isClassified(tx.To) {
        go p.classificationService.ClassifyWalletAddress(context.Background(), tx.To)
    }

    return nil
}
```

### Wallet Repository Integration

Classification data được tự động sync với Wallet entity để duy trì consistency.

## Configuration

```yaml
node_classification:
  enabled: true
  auto_classify: true
  confidence_threshold: 0.6
  batch_size: 100
  cache_ttl: 3600
  blacklist_sources:
    - ofac
    - eu_sanctions
    - custom
  exchange_patterns:
    binance: ["^0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be$"]
    coinbase: ["^0x71660c4005ba85c37ccec55d0c4493e66fe775d3$"]
```

## Roadmap

### Phase 1 (Current)
- ✅ Basic node classification
- ✅ Risk level assessment
- ✅ Pattern-based detection
- ✅ Neo4j integration

### Phase 2 (Next)
- [ ] Machine learning models
- [ ] Real-time monitoring dashboard
- [ ] Advanced cluster analysis
- [ ] API endpoints

### Phase 3 (Future)
- [ ] Cross-chain analysis
- [ ] Advanced ML algorithms
- [ ] Regulatory reporting
- [ ] Third-party integrations

## Troubleshooting

### Common Issues

1. **Classification not updating**: Check cache TTL và background jobs
2. **Low confidence scores**: Review classification rules và patterns
3. **Performance issues**: Enable caching và batch processing

### Debugging

```go
// Enable debug logging
log.SetLevel(log.DebugLevel)

// Check classification details
classification, err := repo.GetClassification(ctx, address)
fmt.Printf("Classification details: %+v\n", classification)
```

## Contributing

Để đóng góp vào hệ thống classification:

1. Thêm node types mới trong `entity/node_types.go`
2. Cập nhật classification rules trong `service/node_classifier.go`
3. Thêm test cases
4. Cập nhật documentation

## License

[License information here]