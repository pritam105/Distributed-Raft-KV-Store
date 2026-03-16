// The in-memory key-value store (the "state machine").
//
// This is just a map[string]string protected by a read-write mutex.
// It is the thing Raft is replicating. The store itself knows nothing
// about Raft — it only exposes Get / Put / Delete operations.
//
// Writes are ONLY allowed through apply.go (after Raft commits an entry).
// Direct writes from clients are rejected if this node is not the leader.
package kv
