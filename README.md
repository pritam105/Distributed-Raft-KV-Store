# Distributed-Raft-KV-Store

## What we have so far

Right now this repo contains a simple single-node key-value store in Go.

It supports:

- upsert a key
- read a key
- delete a key
- optional WAL (write-ahead log)
- optional snapshot persistence
- recovery from persisted snapshot data and WAL replay
- a small HTTP service for local testing

This is the local building block we can later place behind Raft replication and sharding.

## Simple architecture

The current flow is:

1. A client sends an HTTP request.
2. The HTTP server reads the request and calls the KV store.
3. The KV store appends the write to the WAL if WAL is enabled.
4. The KV store updates the in-memory map.
5. The KV store saves the full map as a snapshot if snapshots are enabled.
6. On restart, the service loads the snapshot first and then replays the WAL.

## Important files

### Core KV store

- `kv/store.go`
  This is the in-memory key-value store. It supports `Upsert`, `Get`, `Delete`, and startup recovery from snapshot + WAL.

- `kv/apply.go`
  Small helper that applies a storage entry to the store. This is useful later when Raft hands committed entries to the store.

### Persistence

- `storage/logstore.go`
  Contains the WAL implementation.
  It defines:
  - `Entry`: one operation in the log
  - `WAL`: interface for append/load/close
  - `FileWAL`: real WAL on disk
  - `NoopWAL`: disabled WAL mode

- `storage/snapshot.go`
  Contains snapshot persistence.
  It defines:
  - `SnapshotStore`: interface for save/load
  - `FileSnapshot`: stores the full KV map as JSON on disk
  - `NoopSnapshot`: disabled snapshot mode

- `storage/persistence.go`
  Small configuration wrapper used to open WAL and snapshot persistence.

### HTTP service

- `api/http.go`
  The HTTP API layer. It exposes endpoints for health checks and key operations.

- `cmd/simplekvs/main.go`
  Entry point for running the single-node key-value store as a service.
  It reads environment variables, opens the WAL and snapshot store, restores data, and starts the HTTP server.

### Container setup

- `api/Dockerfile`
  Dockerfile for building and running the simple HTTP key-value store in a container.

## How persistence works

There are two persistence layers right now:

- WAL
  Every write operation is recorded as an append-only log entry.

- Snapshot
  After a successful write, the current full KV map is saved to disk as JSON.

On startup:

1. the latest snapshot is loaded
2. the WAL is replayed
3. the in-memory map is rebuilt to the latest state

This gives us a simple version of true persistence while still keeping WAL-based recovery.

## How to run the KV store locally

Run this from the repo root:

```cmd
go run ./cmd/simplekvs
```

By default it starts with:

- HTTP address: `:8080`
- WAL enabled: `true`
- WAL path: `data/wal.log`
- snapshot enabled: `true`
- snapshot path: `data/snapshot.json`

### Environment variables

- `KVS_ADDR`
  HTTP listen address
  Example: `:8080`

- `KVS_WAL_ENABLED`
  Set to `false` to disable WAL

- `KVS_WAL_PATH`
  Path to WAL file

- `KVS_SNAPSHOT_ENABLED`
  Set to `false` to disable snapshot persistence

- `KVS_SNAPSHOT_PATH`
  Path to snapshot file

Example:

```cmd
set KVS_ADDR=:8080
set KVS_WAL_ENABLED=true
set KVS_WAL_PATH=data/wal.log
set KVS_SNAPSHOT_ENABLED=true
set KVS_SNAPSHOT_PATH=data/snapshot.json
go run ./cmd/simplekvs
```

Example without WAL:

```cmd
set KVS_WAL_ENABLED=false
go run ./cmd/simplekvs
```

Example without snapshot persistence:

```cmd
set KVS_SNAPSHOT_ENABLED=false
go run ./cmd/simplekvs
```

## How to run it as an HTTP service

The service exposes these endpoints:

- `GET /healthz`
- `PUT /v1/keys/{key}`
- `GET /v1/keys/{key}`
- `DELETE /v1/keys/{key}`

Base URL:

```text
http://localhost:8080
```

### Example requests

Create or update a key:

```cmd
curl -X PUT http://localhost:8080/v1/keys/demo -H "Content-Type: application/json" -d "{\"value\":\"hello\"}"
```

Read a key:

```cmd
curl http://localhost:8080/v1/keys/demo
```

Delete a key:

```cmd
curl -X DELETE http://localhost:8080/v1/keys/demo
```

Health check:

```cmd
curl http://localhost:8080/healthz
```

## How to run with Docker

The Dockerfile lives at `api/Dockerfile`.

Build from the repo root:

```cmd
docker build -f api/Dockerfile -t simplekvs .
```

Run the container:

```cmd
docker run --rm -p 8080:8080 -v "%CD%/data:/data" simplekvs
```

What this does:

- starts the container
- maps your machine's port `8080` to the container's port `8080`
- mounts a local `data` folder into `/data` inside the container
- stores the WAL file at `/data/wal.log`
- stores the snapshot file at `/data/snapshot.json`

Run without WAL:

```cmd
docker run --rm -p 8080:8080 -e KVS_WAL_ENABLED=false simplekvs
```

Run without snapshot persistence:

```cmd
docker run --rm -p 8080:8080 -e KVS_SNAPSHOT_ENABLED=false simplekvs
```

## Raft Leader Election

Implements the Raft consensus algorithm's leader election phase for a single
shard group of 3-4 nodes. One leader is elected per term, heartbeats keep
followers stable, and re-election happens automatically when a leader crashes.

### What is implemented

- Follower / Candidate / Leader state machine
- Randomised election timeout (300–600ms) to prevent split votes
- Parallel RequestVote RPCs with majority vote counting
- Leader heartbeats every 100ms via AppendEntries
- Automatic re-election on leader crash
- Term monotonicity across elections
- Node metrics exposed via `/metrics` endpoint
- Gin HTTP server for node-to-node RPC
- Docker-based EC2 deployment via Terraform

### Important files

| File | What it does |
|---|---|
| `raft/state.go` | NodeState enum with atomic transitions |
| `raft/log.go` | LogEntry struct and lastIndexAndTerm |
| `raft/transport.go` | RequestVote + AppendEntries message types |
| `raft/raft.go` | Node struct, metrics, Start/Stop |
| `raft/election.go` | Full state machine and RPC handlers |
| `raft/cluster.go` | In-process transport for tests |
| `raft/election_test.go` | 5 unit tests |
| `rpc/server.go` | Gin HTTP server |
| `rpc/client.go` | HTTP transport for EC2 nodes |
| `cmd/node/main.go` | Binary entry point |

### Run tests locally
```bash
go test ./raft/... -v -timeout 60s
```

### Run a 3-node cluster locally
```bash
# Terminal 1
RAFT_NODE_ID=nodeA \
RAFT_PEERS="nodeB@localhost:7001,nodeC@localhost:7002" \
RAFT_ADDR=:7000 \
go run ./cmd/node

# Terminal 2
RAFT_NODE_ID=nodeB \
RAFT_PEERS="nodeA@localhost:7000,nodeC@localhost:7002" \
RAFT_ADDR=:7001 \
go run ./cmd/node

# Terminal 3
RAFT_NODE_ID=nodeC \
RAFT_PEERS="nodeA@localhost:7000,nodeB@localhost:7001" \
RAFT_ADDR=:7002 \
go run ./cmd/node
```

Check status:
```bash
curl http://localhost:7000/status
curl http://localhost:7001/status
curl http://localhost:7002/status
```

### Deploy to EC2
```bash
cd infrastructure/raft_leader_election_infra/terraform
terraform init
terraform apply -var="key_name=your-key-name"
```
### Check election status on EC2

Once deployed, check which node is leader:
```bash
curl http://<nodeA_ip>:7000/status
curl http://<nodeB_ip>:7000/status
curl http://<nodeC_ip>:7000/status
```

Check metrics:
```bash
curl http://<nodeA_ip>:7000/metrics
curl http://<nodeB_ip>:7000/metrics
curl http://<nodeC_ip>:7000/metrics
```

Simulate a leader crash:
```bash
ssh -i your-key.pem ec2-user@<leader_ip> "sudo systemctl stop raft-node"
```

Watch re-election happen on the remaining nodes:
```bash
curl http://<nodeB_ip>:7000/status
# will show new leader within ~300-600ms
```

Restart the crashed node and watch it rejoin as Follower:
```bash
ssh -i your-key.pem ec2-user@<leader_ip> "sudo systemctl start raft-node"
curl http://<leader_ip>:7000/status
# will show Follower with new higher term
```

Run the full timing experiment:
```bash
python3 infrastructure/raft_leader_election_infra/experiment.py \
  --node-a <nodeA_ip> \
  --node-b <nodeB_ip> \
  --node-c <nodeC_ip> \
  --key ~/.ssh/your-key.pem \
  --rounds 5
```
### What the tests cover

| Test | What it proves |
|---|---|
| `TestElection_3Nodes` | 3 nodes elect exactly one leader |
| `TestElection_4Nodes` | 4 nodes elect exactly one leader |
| `TestElection_LeaderCrash` | Leader dies → new leader elected at higher term |
| `TestElection_FollowerCrash` | Follower dies → leader stays stable |
| `TestElection_TermIncreases` | Term always increases across re-elections |

### EC2 experiment results (us-west-2, 3x t3.micro)

5 rounds of leader crash + re-election:

| Round | Crashed | New Leader | Time |
|---|---|---|---|
| 1 | nodeA | nodeB | 190ms |
| 2 | nodeB | nodeA | 190ms |
| 3 | nodeA | nodeC | 381ms |
| 4 | nodeC | nodeB | 368ms |
| 5 | nodeB | nodeC | 368ms |

Average re-election time: **299ms**

---

## Consistent Hashing and Shard Routing

Distributes keys across independent KV nodes using consistent hashing. A stateless router sits in front of the KV nodes, hashes each key to a shard, and forwards the request to the correct node.

### What is implemented

- FNV-1a consistent hashing — every client and node uses the same deterministic function, no central directory needed
- Stateless shard router — hashes the key, selects the target shard, forwards `GET`/`PUT`/`DELETE` to the node's existing `/v1/keys/{key}` API
- Multi-address fallback per shard — router tries each address in order, forward-compatible with Raft leaders
- Cluster config loaded from environment variables at startup
- CLI client (`cmd/client`) with `get`, `put`, `del` commands — prints the resolved shard ID alongside each result

### Important files

| File | What it does |
|---|---|
| `shard/hashing.go` | `KeyToShard(key, total)` — FNV-1a hash mod total shards |
| `shard/shard_group.go` | `Shard` struct — ID + list of node addresses |
| `shard/router.go` | `Router` — routes and forwards Get/Put/Delete |
| `config/config.go` | `LoadFromEnv()` — builds cluster topology from env vars |
| `cmd/client/main.go` | CLI entry point |
| `functional/sharding_test.go` | Functional tests for hashing, routing, and config |

### Run a 2-shard setup locally

```bash
# Terminal 1 — shard 0
go run ./cmd/simplekvs

# Terminal 2 — shard 1
KVS_ADDR=:8081 KVS_WAL_PATH=data/wal1.log KVS_SNAPSHOT_PATH=data/snap1.json \
go run ./cmd/simplekvs

# Terminal 3 — use the CLI client
export CLIENT_SHARDS_TOTAL=2
export CLIENT_SHARD_0_ADDRS=http://localhost:8080
export CLIENT_SHARD_1_ADDRS=http://localhost:8081

go run ./cmd/client put name alice
go run ./cmd/client put city london
go run ./cmd/client get name
go run ./cmd/client get city
go run ./cmd/client del name
```

Each response prints which shard the key landed on:

```
ok  (shard 1)
ok  (shard 0)
name = alice  (shard 1)
city = london  (shard 0)
deleted  (shard 1)
```

### Environment variables

| Variable | Description | Example |
|---|---|---|
| `CLIENT_SHARDS_TOTAL` | Total number of shards | `2` |
| `CLIENT_SHARD_0_ADDRS` | Comma-separated addresses for shard 0 | `http://localhost:8080,http://localhost:8081` |
| `CLIENT_SHARD_N_ADDRS` | Addresses for shard N | `http://localhost:8090` |

### Run the sharding tests

```bash
go test ./functional/... -v -run "TestHashing|TestRouter|TestConfig" -timeout 30s
```

### What the tests cover

| Test | What it proves |
|---|---|
| `TestHashingIsDeterministic` | Same key always maps to the same shard |
| `TestHashingResultIsInRange` | Hash output is always within `[0, totalShards)` |
| `TestHashingDistributesAcrossTwoShards` | Keys spread across both shards |
| `TestRouterPutAndGet` | Basic write + read through the router |
| `TestRouterDelete` | Key is absent after delete |
| `TestRouterTwoShardsIsolation` | Data on shard 0 is invisible on shard 1 |
| `TestRouterFallbackToNextAddress` | Router skips a dead node and uses the next |
| `TestConfigLoadFromEnv` | Env vars parse into the correct shard topology |
| `TestConfigLoadFromEnvMissingAddr` / `InvalidTotal` | Bad config is rejected |