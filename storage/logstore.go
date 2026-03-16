// Persists the Raft log to disk (append-only file).
//
// Without this, a crashed node forgets every log entry it had and cannot
// safely rejoin the cluster. Every time a new entry is appended to the
// in-memory log it is also written here.
//
// On restart, this file is replayed to rebuild the in-memory log before
// the node participates in any Raft round.
package storage
