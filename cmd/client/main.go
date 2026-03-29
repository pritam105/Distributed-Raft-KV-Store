package main

import (
	"fmt"
	"log"
	"os"

	"distributed-raft-kv-store/config"
	"distributed-raft-kv-store/shard"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage:")
		fmt.Fprintln(os.Stderr, "  client get <key>")
		fmt.Fprintln(os.Stderr, "  client put <key> <value>")
		fmt.Fprintln(os.Stderr, "  client del <key>")
		os.Exit(1)
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	router := shard.NewRouter(cfg.Shards, cfg.TotalShards)

	cmd := os.Args[1]
	key := os.Args[2]

	switch cmd {
	case "get":
		value, found, err := router.Get(key)
		if err != nil {
			log.Fatalf("get failed: %v", err)
		}
		if !found {
			fmt.Printf("key %q not found\n", key)
			os.Exit(1)
		}
		fmt.Printf("%s = %s  (shard %d)\n", key, value, router.ShardFor(key))

	case "put":
		if len(os.Args) < 4 {
			log.Fatal("put requires a value: client put <key> <value>")
		}
		value := os.Args[3]
		if err := router.Put(key, value); err != nil {
			log.Fatalf("put failed: %v", err)
		}
		fmt.Printf("ok  (shard %d)\n", router.ShardFor(key))

	case "del":
		if err := router.Delete(key); err != nil {
			log.Fatalf("delete failed: %v", err)
		}
		fmt.Printf("deleted  (shard %d)\n", router.ShardFor(key))

	default:
		log.Fatalf("unknown command %q — use get / put / del", cmd)
	}
}
