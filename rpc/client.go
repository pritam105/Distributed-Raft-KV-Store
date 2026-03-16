// RPC client: dials a remote node and makes calls to it.
//
// Wraps net/rpc.Dial and provides typed call helpers so the rest of the
// codebase doesn't deal with raw interface{} arguments.
//
// Used by:
//   - raft/transport.go  to send RequestVote / AppendEntries to peers
//   - shard/router.go    to forward client requests to the right leader
//   - cmd/client/main.go to send commands from the CLI tool
package rpc
