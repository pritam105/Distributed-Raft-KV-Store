// The replicated log: data structures and helper operations.
//
// The log is the heart of Raft. Every write command (PUT, DELETE) becomes a
// LogEntry before anything is applied to the KV store. Entries are:
//   - appended locally first
//   - replicated to followers
//   - committed once a majority has them
//   - applied to the KV state machine only after commit
//
// This file defines the LogEntry type and small helpers (e.g. lastLogIndex).
// No networking or timer logic lives here.
package raft
