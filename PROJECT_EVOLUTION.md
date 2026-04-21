# Project Evolution: Distributed Raft KV Store

**CS 6650 — Building Scalable Distributed Systems**  
**Team**: Shreyans Mulkutkar · Kevin Johnson · Pritam Mane

---

## Initial Design

We broke the problem into three vertical slices that could be developed in parallel and composed together:

```
Phase 1 — Foundation          Phase 2 — Consensus           Phase 3 — Scale
─────────────────────         ───────────────────────        ──────────────────────
Single-node KV store          Raft leader election           Raft log replication
WAL + snapshot                RequestVote RPCs               AppendEntries RPCs
HTTP API                      Heartbeats                     KV wired into Raft
SimpleKVS baseline            Failover testing               Sharding + client
                                                             Load experiments
```

The idea was to keep the KV store and Raft consensus loosely coupled so the KV layer could be tested standalone (as SimpleKVS) before being wired into Raft.

---

## Who Worked on What

### Shreyans — KV Store Foundation + Geo Testing

**Built:**
- `kv/store.go` — in-memory map with `Upsert`, `Get`, `Delete`
- `storage/logstore.go` — append-only WAL (`FileWAL` + `NoopWAL`)
- `storage/snapshot.go` — full map serialized to JSON on disk (`FileSnapshot`)
- `kv/apply.go` — `Apply(entry)` that writes to both WAL and in-memory map
- `cmd/simplekvs/` — standalone single-node binary used as the replication experiment baseline
- `api/http.go` — HTTP API (`PUT /v1/keys/{key}`, `GET`, `DELETE`, `GET /healthz`)

**Experiments:**
- Geo-colocated and geo-distributed latency tests — compared Raft cluster performance when nodes are in the same AZ vs spread across regions, measuring how network round-trip time affects consensus latency

---

### Kevin — Leader Election + Failover Testing

**Built:**
- `raft/state.go` — `NodeState` enum (Follower / Candidate / Leader) with atomic transitions
- `raft/election.go` — full Raft state machine: randomized election timeout (300–600ms), parallel `RequestVote` RPCs, majority vote counting, heartbeat loop
- `raft/transport.go` — `RequestVoteArgs/Reply` and initial `AppendEntriesArgs` structs
- `raft/raft.go` — `Node` struct, `Start()` / `Stop()`
- `raft/cluster.go` — in-process transport for unit tests
- `raft/election_test.go` — 5 unit tests covering single leader guarantee, crash + re-election, term monotonicity

**Experiments:**
- Leader election failover experiment — measured re-election time across 5 rounds under live write load. Average re-election time: ~1200ms, zero data loss on committed writes.

---

### Pritam — Raft Log Replication + Sharding + Load Experiments

**Built:**
- `raft/log.go` — extended in-memory Raft log: `entryAt`, `appendEntry`, `truncateAfter`, `entriesFrom`, `lastIndex`
- `raft/replication.go` — `Replicate()`, `replicateToPeer()`, `sendHeartbeatToPeer()`, `advanceCommitIndex()`, `applyCommitted()`
- `raft/election.go` — extended `HandleAppendEntries` to handle log consistency check, entry append, commit advance
- `kv/apply.go` — added `RunApplyLoop(applyCh, store)` goroutine decoupling Raft commit from disk I/O
- `rpc/server.go` — added `/v1/keys/*` HTTP routes; PUT/DELETE go through `node.Replicate()`, GET reads directly from store
- `cmd/node/main.go` — wired full stack: WAL + snapshot + KV store + Raft node + apply loop + HTTP server
- `shard/hashing.go` — `KeyToShard()` using FNV-1a 32-bit (matches Locust experiment script)
- `shard/router.go` — stateless HTTP shard router

**Experiments:**
- Experiment 1: Replication overhead (SimpleKVS vs 3-node Raft, 50 users, same AZ)
- Experiment 2: Horizontal scaling (1-shard vs 2-shard, 50/100 users)

---

## Problems Encountered

### 1. Two half-systems that didn't talk to each other

Initially `cmd/simplekvs` had a working KV store but no Raft, and `cmd/node` had Raft consensus but no KV store. The Raft node would elect a leader and replicate entries but had nowhere to apply them. Wiring the two together required designing the `applyCh` channel interface — Raft commits entries onto the channel, `RunApplyLoop` consumes them and applies to the KV store. This decoupling was the key architectural decision that also solved the disk I/O latency problem (see below).

### 2. Snapshot flushing causing latency growth under load

The original snapshot design wrote the full map to disk on every write. As the store grew, each write triggered a progressively slower disk flush, causing latency to increase monotonically during load tests — making any benchmark meaningless.

Fixed by adding a checkpoint interval (`snapshotInterval = 100`) — only flush every 100 writes. The WAL still appends on every write for durability.

Discovered during experiments: even at 100-write intervals, snapshot growth over a 2.5-minute test caused PUT latency to creep upward. We added `reset_nodes.sh` to clear WAL + snapshot between test runs for clean baselines.

### 3. WAL duplication on node restart

When a follower restarts, its in-memory Raft log is empty. The leader re-sends all committed entries via `AppendEntries`. These entries flow through `applyCh` → `RunApplyLoop` → WAL, causing every entry to be written to the WAL a second time. On the next restart, the WAL replay doubles up again.

Root cause: `lastApplied` (the index up to which entries have been applied) is never persisted to disk. The proper fix is persisting `lastApplied` so a restarting node knows what it has already applied. This was left as a known limitation — it doesn't affect correctness (WAL apply is idempotent), only storage efficiency.

### 4. High term numbers from staggered node startup

Running three nodes locally with `go run` caused each node to compile separately — they started at different times, triggering elections before all nodes were up. Terms reached into the thousands before the cluster stabilized.

Fixed by pre-building with `go build -o bin/node ./cmd/node` and running the binary directly, which is near-instant and allowed all three nodes to start within milliseconds of each other.

---

## Final State vs Initial Design

| Component | Initial Plan | Final State |
|---|---|---|
| KV store | In-memory map | Done — WAL + periodic snapshot |
| Raft election | Leader election only | Done — full state machine |
| Raft replication | Planned | Done — AppendEntries, commit tracking, apply loop |
| Sharding | Planned | Done — FNV-1a consistent hashing, stateless router, CLI client |
| Persistence on restart | Full WAL + snapshot | Partial — WAL duplication bug on follower restart (known limitation) |
| Experiments | 3 planned | 3 completed — replication overhead, horizontal scaling, leader failover |
| Infra | Manual EC2 | Terraform + Docker + Locust automation |
