# Experiment 1: Replication Overhead — Report

**Course**: CS6650 — Distributed Systems  
**Date**: April 16, 2026  
**Infrastructure**: AWS us-east-1, t3.micro EC2 instances, same Availability Zone  
**Load tool**: Locust 

---

## Objective

Measure the latency and throughput cost of Raft consensus replication compared to a single-node key-value store under write-heavy load.

---

## Setup

| Component | Description |
|---|---|
| **SimpleKVS** | Single-node in-memory KV store with WAL + periodic snapshot. Writes flushed to disk synchronously before HTTP response. |
| **Raft cluster** | 3-node Raft cluster (1 leader + 2 followers). Writes replicated to majority before HTTP response; WAL/snapshot flushed asynchronously. |
| **Workload** | 50 concurrent users, 75% PUT / 25% GET, random keys from space of 10,000, ~90 seconds per run |
| **Key range** | `key-0` to `key-9999` |
| **Instance type** | t3.micro (2 vCPU, 1 GB RAM) |
| **Network** | All nodes in same AZ — intra-cluster Raft RPCs add ~1–2ms |

---

## Results

### Request Statistics

| Metric | SimpleKVS | Raft (3-node) |
|---|---|---|
| Total RPS | 540 | **585** |
| PUT avg latency | 67.6 ms | **57.5 ms** |
| PUT p50 | 60 ms | **52 ms** |
| PUT p95 | 110 ms | **97 ms** |
| PUT p99 | 220 ms | 200 ms |
| PUT max | 650 ms | 1200 ms |
| GET avg latency | 38.2 ms | 38.7 ms |
| GET p50 | 37 ms | 37 ms |
| GET p95 | 48 ms | 49 ms |
| PUT failures | 0 | 0 |
| GET failures | 299 (404 — key not found) | 0 |

---

## Analysis

### Raft outperforms SimpleKVS on average latency

Counterintuitively, the Raft cluster achieved higher throughput (585 vs 540 RPS) and lower average write latency (57.5 ms vs 67.6 ms). This is explained by an architectural difference, not Raft being inherently faster:

- **SimpleKVS** flushes the WAL to disk **synchronously** before returning the HTTP response. The request waits for the disk write to complete.
- **Raft** sends the HTTP ack as soon as the majority of nodes have acknowledged the write **in-memory** (Raft log). The WAL and snapshot flush happen asynchronously in a separate goroutine (`RunApplyLoop`) after the response is already sent.

This means Raft trades durability for latency — a simultaneous crash of all 3 nodes before the apply loop flushes would lose the last few committed writes.

### Tail latency is higher for Raft

Raft's maximum PUT latency (1200 ms) is nearly double SimpleKVS (650 ms). These spikes correspond to brief Raft consensus coordination delays — slow follower acknowledgments, network jitter causing retries, or the leader waiting near the heartbeat timeout boundary. These are rare events but raise the tail.

### SimpleKVS latency spike (visible in chart)

The SimpleKVS response time chart shows a brief spike around the 17:04:00 mark. This is a **snapshot flush** — every 100 writes, the full KV map is serialized to JSON and written to disk synchronously on the HTTP handler goroutine. The request that triggers the 100th write pays the full flush cost directly. At 405 RPS this occurs roughly every 250ms.

### GET latency is identical

Both systems return reads in ~37–39 ms at p50. In both implementations, reads are served directly from the in-memory map without going through Raft or WAL — confirming that replication overhead is purely a write-path concern.

### SimpleKVS GET failures (404)

The 299 GET failures are all HTTP 404 (key not found), not server errors. They occur at the start of the test before enough writes have accumulated to fill the key space. This is expected behavior. The Raft test had 0 failures because the nodes had residual data from prior manual testing before the test run.

---

## Conclusions

1. **Raft replication overhead in same-AZ deployments is negligible** at this scale. Intra-cluster RPCs add ~1–2 ms per write round-trip, which is dominated by the WAL flush cost already present in SimpleKVS.

2. **The async apply loop architecture is the dominant factor**. Raft's lower average latency is not due to consensus being fast — it's because disk I/O is decoupled from the HTTP response path. SimpleKVS could match this by making its WAL flush asynchronous.

3. **Raft provides stronger fault tolerance at near-zero throughput cost** in same-AZ setups. The 3-node cluster can lose one node and continue serving reads and writes. SimpleKVS has no such resilience.

4. **Raft's tail latency (p99, max) is the real cost** of consensus — occasional coordination delays that single-node systems never experience.
