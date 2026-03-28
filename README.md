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
