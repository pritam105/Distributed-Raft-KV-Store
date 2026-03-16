// Core Raft node struct and its lifecycle (start, stop).
//
// Defines the central Node type that holds ALL of Raft's state:
//   - currentTerm, votedFor (who I voted for this term)
//   - the replicated log
//   - commitIndex (highest entry known to be committed)
//   - lastApplied (highest entry applied to the KV store)
//   - nextIndex / matchIndex (leader bookkeeping per follower)
//
// This file is the "hub" — other files in this package add behavior to Node.
package raft
