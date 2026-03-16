// Entry point for starting a single database node.
//
// A "node" is one server in a shard group. You run this binary once per
// machine (or container). It reads a config file, knows its own ID, and
// brings up the Raft engine + RPC server for that node.
//
// Think of this as the "main()" that boots one participant in the cluster.
package main
