# Distributed Raft KV Store

A distributed key-value store built in Go with Raft consensus, write-ahead logging, snapshotting, and consistent-hashing-based sharding. Designed for CS6650 with two load-test experiments on AWS EC2.

---

## Project Status

| Component | Status |
|---|---|
| SimpleKVS — single-node KV with WAL + snapshot | Done |
| Raft leader election | Done |
| Raft log replication (full write path) | Done |
| KV store wired into Raft node | Done |
| Consistent hashing + shard router | Done |
| CLI client | Done |
| Experiment 1: Replication overhead (SimpleKVS vs Raft) | **Done — results collected** |
| Experiment 2: Horizontal scaling (1-shard vs 2-shard) | **Pending — infra ready, tests not run yet** |
| Experiment 3: Leader election failover (re-election + data survival) | **Done — results collected** |

---

## Architecture Overview

### End-to-end write flow (Raft node)

```
Client PUT
  → rpc/server.go (HTTP handler)
  → node.Replicate(cmd)               # encodes entry as JSON, appends to in-memory Raft log
      → replicateToPeer() × 2         # sends AppendEntries RPC to both followers in parallel
      → majority ack received
      → advanceCommitIndex()
      → applyCommitted() → applyCh    # pushes committed entry onto buffered channel
  ← HTTP 200 returned to client

(async, separate goroutine)
  → kv.RunApplyLoop() reads from applyCh
  → store.Apply(entry)                # writes to in-memory map + WAL
  → every 100 writes: snapshot flush to disk
```

**Key design point**: the HTTP ack is sent after majority Raft replication (in-memory on peers), NOT after disk flush. This is faster but means a simultaneous crash of all 3 nodes before `RunApplyLoop` processes the entry could lose that write.

### End-to-end read flow

```
Client GET → rpc/server.go → store.Get() → return value
```

Reads bypass Raft entirely — served directly from the leader's in-memory map. This is a **stale read** design: followers could return slightly older values if they lag behind. Acceptable for this project scope.

---

## Repository Layout

```
cmd/
  node/         — Raft node binary (full stack: Raft + KV + WAL + snapshot)
  simplekvs/    — Single-node KV binary (baseline for replication experiment)
  client/       — CLI client with shard-aware routing

raft/
  raft.go       — Node struct, fields: commitIndex, lastApplied, nextIndex, matchIndex, applyCh
  election.go   — Follower/Candidate/Leader state machine, RequestVote, HandleAppendEntries
  replication.go — Replicate(), replicateToPeer(), sendHeartbeatToPeer(), applyCommitted()
  log.go        — In-memory Raft log: entryAt, appendEntry, truncateAfter, entriesFrom
  transport.go  — AppendEntriesArgs/Reply, RequestVoteArgs/Reply structs
  cluster.go    — In-process transport for unit tests

kv/
  store.go      — In-memory map with WAL append + periodic snapshot (every 100 writes)
  apply.go      — Apply(entry) method + RunApplyLoop(applyCh, store) goroutine

storage/
  logstore.go   — WAL: FileWAL (append-only) and NoopWAL
  snapshot.go   — Snapshot: FileSnapshot (full JSON map) and NoopSnapshot
  persistence.go — Config wrapper to open WAL and snapshot

shard/
  hashing.go    — KeyToShard(key, total) using FNV-1a (matches Locust experiment script)
  shard_group.go — Shard struct: ID + list of node addresses
  router.go     — Stateless HTTP router: hashes key → shard → forwards request

config/
  config.go     — LoadFromEnv(): reads CLIENT_SHARDS_TOTAL + CLIENT_SHARD_N_ADDRS

rpc/
  server.go     — Gin HTTP server: /status, /vote, /append, /v1/keys/*
  client.go     — HTTP transport for inter-node Raft RPCs

api/
  http.go       — SimpleKVS HTTP API (used only by cmd/simplekvs)

functional/     — Integration tests for sharding and routing
infrastructure/
  experiments/
    Dockerfile                          — Multi-stage build: produces node + simplekvs binaries
    entrypoint.sh                       — Selects binary via SERVICE_TYPE env var
    replication_overhead/               — Experiment 1
      terraform/                        — AWS infra: 3 Raft nodes + 1 SimpleKVS on t3.micro
      experiment.py                     — Locust: SimpleKVSUser and RaftUser (75% write / 25% read)
      results/                          — PDF reports from completed runs
      restart_nodes.sh                  — SSH into all nodes, pull latest image, restart service
      reset_nodes.sh                    — Clear WAL + snapshot on all nodes for a clean test run
    horizontal_scaling/                 — Experiment 2
      terraform/                        — AWS infra: 6 Raft nodes (2 shards of 3)
      experiment.py                     — Locust: ShardedUser with FNV-1a shard routing
```

---

## Persistence Design

Two persistence layers, both in `storage/`:

- **WAL** — every write is appended as a log entry before the in-memory map is updated
- **Snapshot** — full map serialized to JSON every 100 writes (configurable via `snapshotInterval`)

On startup (`kv.NewStoreFromDisk`):
1. Load snapshot → rebuild map
2. Replay WAL entries after snapshot → bring map to latest state

**Known limitation**: the Raft log is in-memory only. On node restart the Raft log is empty, so the leader re-replicates all committed entries to the restarted follower via `AppendEntries`. The entries are re-applied through the WAL (idempotent), but this causes WAL duplication. Fixing this would require persisting `lastApplied` to disk — out of scope for now.

---

## Running Locally

### Build first (avoids compile-time races between nodes)

```bash
go build -o bin/node ./cmd/node
go build -o bin/simplekvs ./cmd/simplekvs
go build -o bin/client ./cmd/client
```

### 3-node Raft cluster

```bash
# Terminal 1
RAFT_NODE_ID=nodeA \
RAFT_PEERS="nodeB@localhost:8001,nodeC@localhost:8002" \
RAFT_ADDR=:8000 \
RAFT_WAL_PATH=data/nodeA/wal.log \
RAFT_SNAPSHOT_PATH=data/nodeA/snapshot.json \
./bin/node

# Terminal 2
RAFT_NODE_ID=nodeB \
RAFT_PEERS="nodeA@localhost:8000,nodeC@localhost:8002" \
RAFT_ADDR=:8001 \
RAFT_WAL_PATH=data/nodeB/wal.log \
RAFT_SNAPSHOT_PATH=data/nodeB/snapshot.json \
./bin/node

# Terminal 3
RAFT_NODE_ID=nodeC \
RAFT_PEERS="nodeA@localhost:8000,nodeB@localhost:8001" \
RAFT_ADDR=:8002 \
RAFT_WAL_PATH=data/nodeC/wal.log \
RAFT_SNAPSHOT_PATH=data/nodeC/snapshot.json \
./bin/node
```

Find the leader:
```bash
curl http://localhost:8000/status
curl http://localhost:8001/status
curl http://localhost:8002/status
```

Test writes and reads (use the leader's port):
```bash
curl -X PUT http://localhost:8000/v1/keys/name \
  -H "Content-Type: application/json" -d '{"value":"alice"}'

curl http://localhost:8000/v1/keys/name
curl -X DELETE http://localhost:8000/v1/keys/name
```

Test failure recovery — kill the leader and watch re-election:
```bash
# Kill leader process, then check another node
curl http://localhost:8001/status   # new leader elected within ~300-600ms
```

### Shard-aware CLI client

```bash
export CLIENT_SHARDS_TOTAL=2
export CLIENT_SHARD_0_ADDRS=http://localhost:8000
export CLIENT_SHARD_1_ADDRS=http://localhost:8001

./bin/client put name alice
./bin/client put city london
./bin/client get name       # prints: name = alice  (shard 1)
./bin/client del name
```

---

## Tests

```bash
# Raft unit tests (election, replication)
go test ./raft/... -v -timeout 60s

# Sharding + routing functional tests
go test ./functional/... -v -timeout 30s
```

### Raft test coverage

| Test | What it verifies |
|---|---|
| `TestElection_3Nodes` | 3 nodes elect exactly one leader |
| `TestElection_LeaderCrash` | Leader crash → new leader at higher term |
| `TestElection_FollowerCrash` | Follower crash → leader stays stable |
| `TestElection_TermIncreases` | Term always increases across re-elections |

### Sharding test coverage

| Test | What it verifies |
|---|---|
| `TestHashingIsDeterministic` | Same key always maps to same shard |
| `TestHashingDistributesAcrossTwoShards` | Keys spread across both shards |
| `TestRouterPutAndGet` | Write + read through the router |
| `TestRouterTwoShardsIsolation` | Data on shard 0 is not visible on shard 1 |
| `TestRouterFallbackToNextAddress` | Router skips a dead node and uses the next |
| `TestConfigLoadFromEnv` | Env vars parse into correct shard topology |

---

## Experiments

All experiments run on AWS (us-east-1, t3.micro). Locust load tests run from your laptop.

### Experiment 1: Replication Overhead — COMPLETED

**Question**: How much latency does Raft consensus replication add over a single-node store under write-heavy load?

**Setup**: 50 users, 75% PUT / 25% GET, ~90 seconds, same AZ.

| Metric | SimpleKVS | Raft (3-node) |
|---|---|---|
| Total RPS | 540 | 585 |
| PUT avg latency | 67.6ms | 57.5ms |
| PUT p50 | 60ms | 52ms |
| PUT p95 | 110ms | 97ms |
| PUT max | 650ms | 1200ms |
| GET failures | 299 (404 — expected) | 0 |

**Findings**:
- Raft showed lower average and median write latency than SimpleKVS in this run. This is because the Raft apply loop is async — the HTTP ack is sent before the disk flush, while SimpleKVS flushes the WAL synchronously before responding.
- Raft's tail latency (max 1200ms) is higher, reflecting occasional Raft consensus coordination delays.
- Same-AZ intra-cluster RPCs add ~1-2ms network overhead per replication round, which is negligible.

Full reports: `infrastructure/experiments/replication_overhead/results/`

**Helper scripts**:
```bash
# Pull latest image + restart all nodes
./infrastructure/experiments/replication_overhead/restart_nodes.sh ~/Downloads/labsuser.pem

# Clear WAL + snapshot on all nodes before a clean test run
./infrastructure/experiments/replication_overhead/reset_nodes.sh ~/Downloads/labsuser.pem
# Wait ~5s after reset for Raft re-election before starting Locust
```

---

### Experiment 2: Horizontal Scaling — PENDING

**Question**: Does throughput scale linearly when adding a second shard (6 nodes total) vs a single shard (3 nodes)?

**Setup**: 2 shards × 3 Raft nodes each. Locust uses FNV-1a to route keys to the correct shard — same hash function as `shard/hashing.go`.

**Infra**: `infrastructure/experiments/horizontal_scaling/terraform/`

**Steps to run**:

```bash
# 1. Deploy (AWS Academy, us-east-1)
cd infrastructure/experiments/horizontal_scaling/terraform
terraform init
terraform apply -var="key_name=vockey" -var="docker_image=pritammane105/raft-kv:latest"

# 2. Find both shard leaders via status_urls output
# 3. Run 1-shard baseline
CLIENT_SHARDS_TOTAL=1 \
CLIENT_SHARD_0_ADDRS=http://<shard0-leader>:8000 \
locust -f infrastructure/experiments/horizontal_scaling/experiment.py ShardedUser \
  --host http://<shard0-leader>:8000 \
  --users 100 --spawn-rate 10 --run-time 60s --headless --csv results_1shard

# 4. Run 2-shard test
CLIENT_SHARDS_TOTAL=2 \
CLIENT_SHARD_0_ADDRS=http://<shard0-leader>:8000 \
CLIENT_SHARD_1_ADDRS=http://<shard1-leader>:8000 \
locust -f infrastructure/experiments/horizontal_scaling/experiment.py ShardedUser \
  --host http://<shard0-leader>:8000 \
  --users 100 --spawn-rate 10 --run-time 60s --headless --csv results_2shard

# 5. Teardown
terraform destroy -var="key_name=vockey" -var="docker_image=pritammane105/raft-kv:latest"
```

**Expected outcome**: ~2× throughput improvement (RPS) with 2 shards since each shard handles an independent write quorum.

---

## Docker Image

The single image contains both `node` and `simplekvs` binaries. `SERVICE_TYPE` controls which one runs.

```bash
# Build for EC2 (linux/amd64 — required when building on Apple Silicon)
docker buildx build --platform linux/amd64 \
  -f infrastructure/experiments/Dockerfile \
  -t pritammane105/raft-kv:latest --push .
```

Environment variables for the Raft node container:

| Variable | Default | Description |
|---|---|---|
| `SERVICE_TYPE` | `node` | `node` or `simplekvs` |
| `RAFT_NODE_ID` | — | Unique node ID |
| `RAFT_PEERS` | — | `id@host:port,id@host:port` |
| `RAFT_ADDR` | — | Listen address e.g. `0.0.0.0:8000` |
| `RAFT_WAL_PATH` | `data/wal.log` | WAL file path |
| `RAFT_SNAPSHOT_PATH` | `data/snapshot.json` | Snapshot file path |

### Experiment 3: Leader Election & Failover — COMPLETED

**Question**: Does the cluster automatically re-elect a leader under live write load, and is data preserved during failover?

**Setup**: 3 Raft nodes on t3.micro, us-east-1. 20 Locust users, 75% PUT / 25% GET. Leader stopped via AWS CLI mid-load.

| Metric | Value |
|---|---|
| Re-election time (min) | 641ms |
| Re-election time (max) | 1739ms |
| Re-election time (avg) | ~1200ms |
| Total data loss | 0 keys |
| Pre-crash writes survived | 100% |
| Locust RPS baseline | 183 RPS |

**Findings**:
- Zero data loss — Raft replication guarantees committed writes survive leader failure
- Re-election is automatic — cluster self-heals with no manual intervention
- Locust shows clear V-shape: RPS drops to 0 during crash, recovers after re-election
- Leader-only writes enforced — followers return 503 correctly during re-election gap

**Results**: `infrastructure/experiments/leader_election/results/leader_election_failover.png`

**Infra**: `infrastructure/experiments/leader_election/terraform/`