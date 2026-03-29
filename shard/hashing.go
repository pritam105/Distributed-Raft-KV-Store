package shard

import "hash/fnv"

// KeyToShard maps any key string to a shard ID in [0, totalShards).
// Uses FNV-1a: fast, deterministic, uniform distribution.
// Every node and client must call this same function — that is what
// makes routing work without a central directory.
func KeyToShard(key string, totalShards int) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % totalShards
}
