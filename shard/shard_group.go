// Describes a shard group: which nodes belong to it and which shards it owns.
//
// A shard group is a set of nodes (e.g. NodeA, NodeB, NodeC) that run one
// Raft cluster together and collectively own a range of shard IDs.
//
// This file holds the data types and helper methods for querying that
// configuration — e.g. "give me the peer addresses for shard group 2".
//
// For this project the mapping is static (defined in config.yaml at startup).
package shard
