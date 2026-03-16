// Periodic snapshots of the KV state machine.
//
// Over time the log grows unboundedly. Snapshotting solves this:
//   - serialize the entire KV map to disk at a given log index
//   - truncate all log entries before that index (they are now redundant)
//
// On restart, load the snapshot first (instant KV state recovery), then
// replay only the log entries that came after the snapshot.
//
// This is optional for week 1-3 but essential for week 4 crash recovery.
package storage
