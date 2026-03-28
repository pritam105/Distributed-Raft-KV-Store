# Distributed-Raft-KV-Store

## What we have so far

Right now this repo contains a simple single-node key-value store in Go.

It supports:

- upsert a key
- read a key
- delete a key
- optional WAL (write-ahead log)
- recovery from WAL on restart
- a small HTTP service for testing

This is the local building block we can later place behind Raft replication and sharding.

## Simple architecture

The current flow is:

1. A client sends an HTTP request.
2. The HTTP server reads the request and calls the KV store.
3. The KV store updates in-memory state.
4. If WAL is enabled, the operation is first appended to the WAL file.
5. On restart, the store replays the WAL and rebuilds the latest state.

## Important files

### Core KV store

- `kv/store.go`
  This is the in-memory key-value store. It supports `Upsert`, `Get`, `Delete`, and replay-based startup.

- `kv/apply.go`
  Small helper that applies a storage entry to the store. This is useful because later Raft can hand committed entries to the store in a consistent format.

### WAL and persistence

- `storage/logstore.go`
  Contains the WAL implementation.
  It defines:
  - `Entry`: one operation in the log
  - `WAL`: interface for append/load/close
  - `FileWAL`: real WAL on disk
  - `NoopWAL`: disabled WAL mode

- `storage/persistence.go`
  Small configuration wrapper used to open either a real WAL or a no-op WAL.

### HTTP service

- `api/http.go`
  The HTTP API layer. It exposes endpoints for health checks and key operations.

- `cmd/simplekvs/main.go`
  Entry point for running the single-node key-value store as a service.
  It reads environment variables, opens the WAL, restores data, and starts the HTTP server.

### Container setup

- `api/Dockerfile`
  Dockerfile for building and running the simple HTTP key-value store in a container.

### Tests

- `kv/store_test.go`
  Unit tests for basic KV behavior and replay from WAL.

- `storage/logstore_test.go`
  Unit tests for WAL behavior.

- `api/http_test.go`
  Unit tests for the HTTP API.

## How to run the KV store locally

Run this from the repo root:

```cmd
go run ./cmd/simplekvs
```

By default it starts:

- HTTP address: `:8080`
- WAL enabled: `true`
- WAL path: `data/wal.log`

### Environment variables

- `KVS_ADDR`
  HTTP listen address
  Example: `:8080`

- `KVS_WAL_ENABLED`
  Set to `false` to disable WAL

- `KVS_WAL_PATH`
  Path to WAL file

Example:

```cmd
set KVS_ADDR=:8080
set KVS_WAL_ENABLED=true
set KVS_WAL_PATH=data/wal.log
go run ./cmd/simplekvs
```

Example without WAL:

```cmd
set KVS_WAL_ENABLED=false
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

Run without WAL:

```cmd
docker run --rm -p 8080:8080 -e KVS_WAL_ENABLED=false simplekvs
```

## What this is not yet

This is not yet the full distributed system from the proposal.

It does not yet include:

- Raft replication
- leader election
- shard routing
- multi-node deployment
- distributed failure handling

Right now it is the simplest working storage layer and HTTP interface we can use as a foundation for the distributed version later.
