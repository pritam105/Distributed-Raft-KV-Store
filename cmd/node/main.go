package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"distributed-raft-kv-store/raft"
	"distributed-raft-kv-store/rpc"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parsePeers(raw string) ([]string, rpc.PeerMap) {
	ids := []string{}
	addrs := rpc.PeerMap{}

	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// format: nodeA@10.0.0.1:7000
		parts := strings.SplitN(entry, "@", 2)
		if len(parts) != 2 {
			log.Fatalf("bad peer entry %q — want nodeID@host:port", entry)
		}
		id, hostport := parts[0], parts[1]
		ids = append(ids, id)
		addrs[id] = "http://" + hostport
	}
	return ids, addrs
}

func main() {
	nodeID := getEnv("RAFT_NODE_ID", "nodeA")
	peersRaw := getEnv("RAFT_PEERS", "")
	addr := getEnv("RAFT_ADDR", ":7000")

	log.Printf("[%s] starting on %s", nodeID, addr)

	peerIDs, peerAddrs := parsePeers(peersRaw)
	log.Printf("[%s] peers: %v", nodeID, peerIDs)

	transport := rpc.NewHTTPTransport(peerAddrs)
	node := raft.NewNode(nodeID, peerIDs, transport)
	node.Start()
	defer node.Stop()

	server := rpc.NewServer(node)

	log.Printf("[%s] listening on %s", nodeID, addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		log.Fatalf("[%s] server error: %v", nodeID, err)
	}
}
