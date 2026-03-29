package shard

// Shard represents one Raft cluster that owns a contiguous range of shard IDs.
// Addrs holds the HTTP addresses of all nodes in that cluster.
// The router tries them in order until one responds (the leader will).
type Shard struct {
	ID    int
	Addrs []string // e.g. ["http://localhost:7000", "http://localhost:7001", "http://localhost:7002"]
}
