// Shared request/response types for all RPCs.
//
// Go's net/rpc requires that RPC argument and reply types be exported structs.
// Centralizing them here means both the server and client import one package
// instead of creating circular dependencies between raft/ and rpc/.
//
// Examples of what lives here:
//   - GetArgs / GetReply
//   - PutArgs / PutReply
//   - DeleteArgs / DeleteReply
package rpc
