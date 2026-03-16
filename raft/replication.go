// Log replication and heartbeat logic (leader side).
//
// Once a node is leader it must:
//   - send periodic heartbeats to all followers (empty AppendEntries RPCs)
//     so they don't time out and start a new election
//   - when a client write comes in, append it to the local log and then
//     replicate it to all followers via AppendEntries RPCs
//   - after a majority of nodes confirm the entry, mark it committed
//
// Also contains the AppendEntries RPC handler (follower side), which checks
// log consistency and appends new entries when valid.
package raft
