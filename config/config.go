// Loads and exposes cluster configuration.
//
// Reads a config file (YAML or JSON) that describes the entire cluster:
//   - list of shard groups and their member node addresses
//   - total number of shards
//   - this node's own ID and listen address
//   - storage paths (where to write log and snapshot files)
//
// Everything else in the codebase reads from this package rather than
// hardcoding addresses or file paths.
package config
