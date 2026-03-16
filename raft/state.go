// State transitions: Follower → Candidate → Leader (and back).
//
// Centralizes the rules for moving between roles so the transitions are
// never scattered across files. For example:
//   - becomeFollower: reset votedFor, stop heartbeat loop
//   - becomeLeader:   initialize nextIndex/matchIndex, start heartbeat loop
//
// Also contains the "apply loop" — a background goroutine that watches
// commitIndex and, whenever it advances, sends newly committed log entries
// to the KV state machine via a channel.
package raft
