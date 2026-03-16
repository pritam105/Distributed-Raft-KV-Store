// Bridges Raft and the KV store.
//
// Listens on the channel that raft/state.go writes committed entries into.
// For each committed LogEntry it receives, it calls the appropriate
// store.Put / store.Delete operation.
//
// This is what makes the replication "real": the log entry has been agreed
// upon by the cluster, so now it is safe to mutate the actual data.
package kv
