// Network transport: how this Raft node calls RPCs on other nodes.
//
// Wraps Go's net/rpc so the rest of the Raft code can call
// peer.RequestVote(...) or peer.AppendEntries(...) without caring about
// connection management, retries, or timeouts.
//
// Keeping transport here means the core Raft logic (election.go, replication.go)
// stays testable — you can swap in a fake transport for unit tests.
package raft
