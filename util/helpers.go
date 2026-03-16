// Small utility functions with no better home.
//
// Candidates for this file:
//   - randomized election timeout generator (used by election.go)
//   - min/max helpers for index arithmetic
//   - encoding helpers (e.g. serialize a struct to bytes for disk writes)
//
// Keep this file small. If any helper grows into a real abstraction,
// move it to its own file or package.
package util
