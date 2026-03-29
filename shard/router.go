package shard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Router hashes each key to the correct shard and forwards HTTP requests
// to the nodes in that shard. It tries each address in order until one
// succeeds — the Raft leader will accept writes; followers will refuse.
type Router struct {
	shards      []*Shard
	totalShards int
	client      *http.Client
}

func NewRouter(shards []*Shard, totalShards int) *Router {
	return &Router{
		shards:      shards,
		totalShards: totalShards,
		client:      &http.Client{Timeout: 2 * time.Second},
	}
}

// Get fetches a value by key from the correct shard.
func (r *Router) Get(key string) (string, bool, error) {
	s := r.route(key)

	for _, addr := range s.Addrs {
		resp, err := r.client.Get(addr + "/v1/keys/" + key)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return "", false, nil
		}
		if resp.StatusCode != http.StatusOK {
			continue
		}

		var body struct {
			Value string `json:"value"`
			Found bool   `json:"found"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return "", false, err
		}
		return body.Value, body.Found, nil
	}

	return "", false, fmt.Errorf("shard %d: no node responded", s.ID)
}

// Put writes a key-value pair to the correct shard.
func (r *Router) Put(key, value string) error {
	s := r.route(key)

	payload, _ := json.Marshal(map[string]string{"value": value})

	for _, addr := range s.Addrs {
		req, _ := http.NewRequest(http.MethodPut, addr+"/v1/keys/"+key, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := r.client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return nil
		}
	}

	return fmt.Errorf("shard %d: no node accepted the write", s.ID)
}

// Delete removes a key from the correct shard.
func (r *Router) Delete(key string) error {
	s := r.route(key)

	for _, addr := range s.Addrs {
		req, _ := http.NewRequest(http.MethodDelete, addr+"/v1/keys/"+key, nil)

		resp, err := r.client.Do(req)
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
			return nil
		}
	}

	return fmt.Errorf("shard %d: no node accepted the delete", s.ID)
}

// ShardFor returns which shard ID a key maps to. Useful for debugging.
func (r *Router) ShardFor(key string) int {
	return KeyToShard(key, r.totalShards)
}

func (r *Router) route(key string) *Shard {
	id := KeyToShard(key, r.totalShards)
	return r.shards[id]
}
