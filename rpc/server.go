// RPC server: registers and exposes this node's methods over the network.
//
// Uses Go's net/rpc package to listen on a TCP port and serve incoming calls.
// Other nodes call Raft RPCs (RequestVote, AppendEntries) through this server.
// Clients call KV RPCs (Get, Put, Delete) through this server as well.
//
// One server instance runs per node process.
package rpc
