package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"distributed-raft-kv-store/shard"
)

// Config describes the full cluster topology seen by the client / router.
type Config struct {
	TotalShards int
	Shards      []*shard.Shard
}

// LoadFromEnv builds a Config from environment variables.
//
// Required vars:
//   CLIENT_SHARDS_TOTAL=2
//   CLIENT_SHARD_0_ADDRS=http://localhost:7000,http://localhost:7001,http://localhost:7002
//   CLIENT_SHARD_1_ADDRS=http://localhost:7100,http://localhost:7101,http://localhost:7102
func LoadFromEnv() (*Config, error) {
	totalStr := os.Getenv("CLIENT_SHARDS_TOTAL")
	if totalStr == "" {
		totalStr = "1"
	}

	total, err := strconv.Atoi(totalStr)
	if err != nil || total < 1 {
		return nil, fmt.Errorf("CLIENT_SHARDS_TOTAL must be a positive integer, got %q", totalStr)
	}

	shards := make([]*shard.Shard, total)
	for i := 0; i < total; i++ {
		key := fmt.Sprintf("CLIENT_SHARD_%d_ADDRS", i)
		raw := os.Getenv(key)
		if raw == "" {
			return nil, fmt.Errorf("missing env var %s", key)
		}

		addrs := []string{}
		for _, a := range strings.Split(raw, ",") {
			a = strings.TrimSpace(a)
			if a != "" {
				addrs = append(addrs, a)
			}
		}

		if len(addrs) == 0 {
			return nil, fmt.Errorf("%s is empty", key)
		}

		shards[i] = &shard.Shard{ID: i, Addrs: addrs}
	}

	return &Config{TotalShards: total, Shards: shards}, nil
}
