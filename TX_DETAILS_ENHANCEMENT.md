# TX_Details Enhancement for ERC20 Relationships

## 📋 Overview

Đã áp dụng **TX_Details approach** (giống như `SENT_TO` relationship) cho **tất cả ERC20 relationships**, cho phép tracking chi tiết từng transaction thay vì chỉ aggregate data.

## 🔧 Changes Implemented

### 1. **Entity Enhancement**

**File**: `internal/domain/entity/erc20.go`

```go
type ERC20TransferRelationship struct {
    // ... existing fields ...
    MethodSignature  string `json:"method_signature"`  // ✅ NEW FIELD
    // ... other fields ...
}
```

### 2. **Neo4j Repository Enhancement**

**File**: `internal/infrastructure/database/neo4j_erc20_repository.go`

#### TX_Details Format
```go
// Enhanced format: "hash:value:timestamp:interaction_type:method_signature"
txDetail := fmt.Sprintf("%s:%s:%s:%s:%s",
    rel.TxHash,
    rel.Value,
    timestampStr,
    string(rel.InteractionType),
    rel.MethodSignature)
```

#### Cypher Queries Updated
**All relationships now include**:
```cypher
ON CREATE SET
    r.tx_details = [rel.tx_detail]    -- Create array with first transaction
ON MATCH SET
    r.tx_details = CASE
        WHEN r.tx_details IS NULL THEN [rel.tx_detail]
        ELSE r.tx_details + rel.tx_detail    -- Append to existing array
    END
```

### 3. **Service Layer Enhancement**

**File**: `internal/application/service/indexing_service.go`

```go
relationship := &entity.ERC20TransferRelationship{
    // ... existing fields ...
    MethodSignature:  transfer.MethodSignature,  // ✅ NEW FIELD
    // ... other fields ...
}
```

## 📊 Supported Relationship Types

All relationship types now support `tx_details`:

| Relationship Type | Purpose | TX_Details Support |
|-------------------|---------|-------------------|
| `ERC20_TRANSFER` | Token transfers | ✅ |
| `ERC20_APPROVAL` | Token approvals | ✅ |
| `DEX_SWAP` | DEX trading | ✅ |
| `LIQUIDITY_OPERATION` | Add/Remove liquidity | ✅ |
| `DEFI_OPERATION` | Deposit/Withdraw | ✅ |
| `MULTICALL_OPERATION` | Multicall transactions | ✅ |
| `ETH_TRANSFER` | ETH transfers | ✅ |
| `CONTRACT_INTERACTION` | Unknown contracts | ✅ |

## 🎯 Benefits

### **1. Detailed Transaction Tracking**
```cypher
MATCH ()-[r:ERC20_TRANSFER]->()
RETURN r.tx_details
```
**Result**: Array of all individual transactions in format:
```
["0xabc123:1000000000000000000:2025-06-26T18:00:00Z:TRANSFER:a9059cbb",
 "0xdef456:2000000000000000000:2025-06-26T18:30:00Z:TRANSFER:a9059cbb"]
```

### **2. Aggregated Data for Performance**
```cypher
MATCH ()-[r:ERC20_TRANSFER]->()
RETURN r.total_value, r.tx_count, r.first_tx, r.last_tx
```

### **3. Easy Transaction Investigation**
```cypher
MATCH ()-[r:DEX_SWAP]->()
UNWIND r.tx_details as detail
WITH split(detail, ":") as parts
RETURN parts[0] as tx_hash, parts[3] as interaction_type, parts[4] as method_sig
```

## 🔍 Query Examples

### Find All Transactions for a Relationship
```cypher
MATCH (from:Wallet)-[r:ERC20_TRANSFER]->(to:Wallet)
WHERE from.address = "0x1111111111111111111111111111111111111111"
UNWIND r.tx_details as detail
WITH split(detail, ":") as parts
RETURN parts[0] as tx_hash,     // Transaction hash
       parts[1] as value,       // Transaction value
       parts[2] as timestamp,   // Transaction timestamp
       parts[3] as type,        // Interaction type
       parts[4] as method       // Method signature
```

### Aggregate Analysis with Transaction Details
```cypher
MATCH ()-[r:DEX_SWAP]->()
RETURN r.total_value,                    // Total volume
       r.tx_count,                       // Number of swaps
       size(r.tx_details) as detail_count, // Verify count
       r.tx_details[0] as first_swap,    // First swap details
       r.tx_details[-1] as last_swap     // Last swap details
```

### Find High-Activity Relationships
```cypher
MATCH ()-[r]->()
WHERE exists(r.tx_details) AND size(r.tx_details) > 10
RETURN type(r) as relationship_type,
       size(r.tx_details) as transaction_count,
       r.total_value
ORDER BY transaction_count DESC
```

## 🔄 Migration Path

### For Existing Data
Relationships created before this enhancement will have:
- ✅ `total_value`, `tx_count`, `first_tx`, `last_tx` (aggregated data)
- ❌ Missing `tx_details` (will be `NULL`)

### For New Data
All new relationships will have:
- ✅ Complete aggregated data
- ✅ Detailed `tx_details` array
- ✅ Enhanced tracking capabilities

## 🚀 Usage in Applications

### TypeScript/JavaScript Frontend
```typescript
interface ERC20Relationship {
  total_value: string;
  tx_count: number;
  first_tx: string;
  last_tx: string;
  tx_details: string[];  // Array of detailed transaction strings
}

// Parse transaction details
function parseTransactionDetail(detail: string) {
  const [hash, value, timestamp, type, methodSig] = detail.split(':');
  return { hash, value, timestamp, type, methodSig };
}
```

### Cypher Analysis Queries
```cypher
// Find most active DEX traders
MATCH (wallet:Wallet)-[r:DEX_SWAP]->(dex)
WHERE size(r.tx_details) > 5
RETURN wallet.address, count(r) as dex_relationships,
       sum(toInteger(r.tx_count)) as total_swaps
ORDER BY total_swaps DESC;

// Analyze trading patterns
MATCH ()-[r:DEX_SWAP]->()
UNWIND r.tx_details as detail
WITH split(detail, ":") as parts
RETURN parts[4] as method_signature, count(*) as usage_count
ORDER BY usage_count DESC;
```

## ✅ Implementation Status

- ✅ **Entity Layer**: Added `MethodSignature` field
- ✅ **Repository Layer**: All relationship types support `tx_details`
- ✅ **Service Layer**: Enhanced data preparation
- ✅ **Database Schema**: Auto-creates `tx_details` arrays
- ✅ **Backwards Compatibility**: Existing aggregated data preserved
- ✅ **Enhanced Format**: Includes interaction type & method signature

## 🎯 Next Steps

1. **Test with Real Data**: Run indexer to verify tx_details creation
2. **Frontend Integration**: Update UI to display transaction details
3. **Analytics Queries**: Create dashboard queries using tx_details
4. **Performance Monitoring**: Monitor query performance with tx_details
5. **Data Export**: Implement export functions for detailed analysis

---

**Note**: This enhancement provides the **best of both worlds** - fast aggregated queries for dashboards and detailed transaction tracking for investigations, following the proven pattern of `SENT_TO` relationships.