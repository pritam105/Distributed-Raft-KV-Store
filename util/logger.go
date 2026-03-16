// Structured logger used across the whole project.
//
// Wraps Go's standard log package (or a lightweight library) to add:
//   - a node ID prefix on every line so logs from different nodes are easy
//     to tell apart when running multiple nodes locally
//   - log levels (DEBUG / INFO / WARN / ERROR)
//
// Every other package imports this instead of calling log.Println directly.
package util
