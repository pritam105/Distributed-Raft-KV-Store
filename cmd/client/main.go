// Entry point for the command-line client tool.
//
// This is a simple CLI that a human (or test script) uses to send GET/PUT/DELETE
// commands to the cluster. It talks to the shard router, which figures out
// which shard group owns the key and forwards the request there.
//
// Not part of the cluster itself — just a driver to interact with it.
package main
