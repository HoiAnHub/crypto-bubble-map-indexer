// Debug queries to check current state of Neo4J database

// 1. Check all relationship types in database
CALL db.relationshipTypes() YIELD relationshipType
RETURN relationshipType;

// 2. Check all property keys in database
CALL db.propertyKeys() YIELD propertyKey
RETURN propertyKey;

// 3. Check all node labels
CALL db.labels() YIELD label
RETURN label;

// 4. Count all relationships by type
MATCH ()-[r]->()
RETURN type(r) as relationship_type, count(r) as count
ORDER BY count DESC;

// 5. Check if ERC20 related nodes exist
MATCH (n:ERC20Contract)
RETURN count(n) as erc20_contracts;

// 6. Check if ERC20_TRANSFER relationships exist
MATCH ()-[r:ERC20_TRANSFER]->()
RETURN count(r) as erc20_transfers;

// 7. Check recent transactions to see what data is coming in
MATCH (w1:Wallet)-[r:SENT_TO]->(w2:Wallet)
RETURN w1.address, w2.address, r.total_value, r.tx_count, r.last_tx
ORDER BY r.last_tx DESC
LIMIT 10;

// 8. Check if there are any transactions with data field
MATCH ()-[r:SENT_TO]->()
WHERE exists(r.tx_details)
RETURN r.tx_details[0] as sample_tx_detail
LIMIT 5;