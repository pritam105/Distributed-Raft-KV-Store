package main

import (
	"log"
	"net/http"
	"os"

	"distributed-raft-kv-store/api"
	"distributed-raft-kv-store/kv"
	"distributed-raft-kv-store/storage"
)

func main() {
	addr := getEnv("KVS_ADDR", ":8080")
	walEnabled := getEnv("KVS_WAL_ENABLED", "true") != "false"
	walPath := getEnv("KVS_WAL_PATH", "data/wal.log")
	snapshotEnabled := getEnv("KVS_SNAPSHOT_ENABLED", "true") != "false"
	snapshotPath := getEnv("KVS_SNAPSHOT_PATH", "data/snapshot.json")

	wal, err := storage.OpenWAL(storage.Config{
		Enabled: walEnabled,
		Path:    walPath,
	})
	if err != nil {
		log.Fatalf("open WAL: %v", err)
	}
	defer wal.Close()

	snapshot, err := storage.OpenSnapshot(storage.Config{
		Enabled: snapshotEnabled,
		Path:    snapshotPath,
	})
	if err != nil {
		log.Fatalf("open snapshot: %v", err)
	}

	store, err := kv.NewStoreFromDisk(wal, snapshot)
	if err != nil {
		log.Fatalf("restore store from disk: %v", err)
	}

	server := api.NewServer(store)

	log.Printf("simplekvs listening on %s (wal_enabled=%t wal_path=%s snapshot_enabled=%t snapshot_path=%s)", addr, walEnabled, walPath, snapshotEnabled, snapshotPath)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
