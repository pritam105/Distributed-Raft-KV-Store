// Persists Raft's critical "hard state": currentTerm and votedFor.
//
// Raft correctness depends on these two values surviving crashes.
// Example: if a node voted for candidate X in term 5, crashed, and forgot —
// it might vote for Y in the same term after restart, breaking the
// "at most one vote per term" guarantee.
//
// This file reads/writes a tiny metadata file (separate from the log file).
package storage
