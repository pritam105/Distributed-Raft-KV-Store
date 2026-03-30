package functional

import (
	"net/http"
	"net/http/httptest"
	"testing"

	apiPkg "distributed-raft-kv-store/api"
	"distributed-raft-kv-store/config"
	"distributed-raft-kv-store/kv"
	"distributed-raft-kv-store/shard"
	"distributed-raft-kv-store/storage"
)

func newShardServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := kv.NewStore(storage.NewNoopWAL(), storage.NewNoopSnapshot())
	return httptest.NewServer(apiPkg.NewServer(store).Handler())
}

func newRouter(t *testing.T, servers ...*httptest.Server) *shard.Router {
	t.Helper()
	shards := make([]*shard.Shard, len(servers))
	for i, s := range servers {
		shards[i] = &shard.Shard{ID: i, Addrs: []string{s.URL}}
	}
	return shard.NewRouter(shards, len(shards))
}

// ---------------------------------------------------------------------------
// Hashing
// ---------------------------------------------------------------------------

func TestHashingIsDeterministic(t *testing.T) {
	for i := 0; i < 50; i++ {
		if shard.KeyToShard("user:42", 8) != shard.KeyToShard("user:42", 8) {
			t.Fatal("KeyToShard returned different results for the same input")
		}
	}
}

func TestHashingResultIsInRange(t *testing.T) {
	for _, key := range []string{"a", "b", "user:1", "city:x"} {
		got := shard.KeyToShard(key, 4)
		if got < 0 || got >= 4 {
			t.Fatalf("key %q mapped to shard %d, out of range [0,4)", key, got)
		}
	}
}

func TestHashingDistributesAcrossTwoShards(t *testing.T) {
	seen := map[int]bool{}
	for _, key := range []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"} {
		seen[shard.KeyToShard(key, 2)] = true
	}
	if len(seen) < 2 {
		t.Fatal("all keys landed on the same shard, expected distribution across both")
	}
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

func TestRouterPutAndGet(t *testing.T) {
	s0 := newShardServer(t)
	defer s0.Close()
	router := newRouter(t, s0)

	if err := router.Put("name", "alice"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	val, found, err := router.Get("name")
	if err != nil || !found || val != "alice" {
		t.Fatalf("Get: err=%v found=%v val=%q", err, found, val)
	}
}

func TestRouterDelete(t *testing.T) {
	s0 := newShardServer(t)
	defer s0.Close()
	router := newRouter(t, s0)

	_ = router.Put("city", "london")
	if err := router.Delete("city"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, found, _ := router.Get("city")
	if found {
		t.Fatal("expected key to be absent after delete")
	}
}

func TestRouterTwoShardsIsolation(t *testing.T) {
	s0, s1 := newShardServer(t), newShardServer(t)
	defer s0.Close()
	defer s1.Close()
	router := newRouter(t, s0, s1)

	// find one key per shard
	var key0, key1 string
	for _, k := range []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"} {
		if router.ShardFor(k) == 0 && key0 == "" {
			key0 = k
		}
		if router.ShardFor(k) == 1 && key1 == "" {
			key1 = k
		}
	}
	if key0 == "" || key1 == "" {
		t.Skip("could not find keys for both shards")
	}

	_ = router.Put(key0, "v0")
	_ = router.Put(key1, "v1")

	// key0 must not be visible on shard 1's server and vice versa
	getJSON(t, s1.URL+"/v1/keys/"+key0, http.StatusNotFound)
	getJSON(t, s0.URL+"/v1/keys/"+key1, http.StatusNotFound)
}

func TestRouterFallbackToNextAddress(t *testing.T) {
	good := newShardServer(t)
	defer good.Close()

	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	dead.Close()

	router := shard.NewRouter([]*shard.Shard{{ID: 0, Addrs: []string{dead.URL, good.URL}}}, 1)

	if err := router.Put("key", "ok"); err != nil {
		t.Fatalf("Put should have fallen back to healthy node: %v", err)
	}
	val, found, _ := router.Get("key")
	if !found || val != "ok" {
		t.Fatalf("Get after fallback: found=%v val=%q", found, val)
	}
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

func TestConfigLoadFromEnv(t *testing.T) {
	t.Setenv("CLIENT_SHARDS_TOTAL", "2")
	t.Setenv("CLIENT_SHARD_0_ADDRS", "http://localhost:8080,http://localhost:8081")
	t.Setenv("CLIENT_SHARD_1_ADDRS", "http://localhost:8090")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.TotalShards != 2 || len(cfg.Shards) != 2 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if len(cfg.Shards[0].Addrs) != 2 {
		t.Fatalf("shard 0 should have 2 addrs, got %d", len(cfg.Shards[0].Addrs))
	}
}

func TestConfigLoadFromEnvMissingAddr(t *testing.T) {
	t.Setenv("CLIENT_SHARDS_TOTAL", "2")
	t.Setenv("CLIENT_SHARD_0_ADDRS", "http://localhost:8080")
	t.Setenv("CLIENT_SHARD_1_ADDRS", "")

	if _, err := config.LoadFromEnv(); err == nil {
		t.Fatal("expected error for missing shard address")
	}
}

func TestConfigLoadFromEnvInvalidTotal(t *testing.T) {
	t.Setenv("CLIENT_SHARDS_TOTAL", "notanumber")

	if _, err := config.LoadFromEnv(); err == nil {
		t.Fatal("expected error for invalid CLIENT_SHARDS_TOTAL")
	}
}
