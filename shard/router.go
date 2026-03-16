// Shard router: receives client requests and forwards them to the right shard group.
//
// The router is the single entry point for clients. It:
//   1. hashes the key to find the target shardID  (uses hashing.go)
//   2. looks up which shard group owns that shardID  (uses shard_group.go config)
//   3. forwards the RPC to the leader of that group
//
// The router itself stores no data. It is purely a request dispatcher.
// In this project the configuration is static, so there is no rebalancing logic.
package shard
