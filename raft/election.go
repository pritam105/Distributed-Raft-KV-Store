// Leader election logic.
//
// Raft requires exactly one leader at a time. This file handles:
//   - the election timer (if no heartbeat arrives in time, start an election)
//   - transitioning to Candidate, incrementing the term, voting for self
//   - sending RequestVote RPCs to all peers
//   - counting votes and becoming Leader if a majority responds yes
//
// Why a separate file: election is a distinct phase of Raft with its own
// RPC type (RequestVote) and timeout logic. Keeping it here avoids bloating raft.go.
package raft
