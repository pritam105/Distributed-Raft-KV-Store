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

	wal, err := storage.OpenWAL(storage.Config{
		Enabled: walEnabled,
		Path:    walPath,
	})
	if err != nil {
		log.Fatalf("open WAL: %v", err)
	}
	defer wal.Close()

	store, err := kv.NewStoreFromWAL(wal)
	if err != nil {
		log.Fatalf("restore store from WAL: %v", err)
	}

	server := api.NewServer(store)

	log.Printf("simplekvs listening on %s (wal_enabled=%t wal_path=%s)", addr, walEnabled, walPath)
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
