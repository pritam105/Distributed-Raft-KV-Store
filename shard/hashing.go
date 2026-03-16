// Consistent hashing: maps a key to a shard ID.
//
// Given a key string, produces a number that deterministically picks
// which shard group owns that key. Every node and every client must
// agree on this mapping — that is what makes routing possible without
// a central directory.
//
// Example: hash("user:42") → shardID 3 → ShardGroup3
package shard
