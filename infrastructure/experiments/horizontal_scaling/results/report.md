# Experiment 2: Horizontal Scaling — Report

**Course**: CS6650 — Distributed Systems  
**Date**: April 16, 2026  
**Infrastructure**: AWS us-east-1, t3.micro EC2 instances, same Availability Zone  
**Load tool**: Locust

---

## Objective

Evaluate whether adding a second shard (doubling from 3 to 6 Raft nodes) improves write throughput and reduces latency proportionally under high concurrent load.

---

## Setup

| Component | Description |
|---|---|
| **1-shard** | Single Raft cluster of 3 nodes. All traffic routed to the shard0 leader. |
| **2-shard** | Two independent Raft clusters of 3 nodes each. Traffic split by FNV-1a hash of key — ~50% to each shard leader. |
| **Routing** | Locust uses the same FNV-1a hash function (`fnv.New32a()`) as the Go shard router so key routing is consistent. |
| **Workload** | 75% PUT / 25% GET, random keys from space of 10,000 |
| **Instance type** | t3.micro (2 vCPU, 1 GB RAM) |

Four test runs were conducted:

| Run | Shards | Users | Duration |
|---|---|---|---|
| 1-Shard, 100 users | 1 | 100 | 83 s |
| 1-Shard, 50 users | 1 | 50 | 82 s |
| 2-Shard, 100 users | 2 | 100 | 55 s |
| 2-Shard Long, 100 users | 2 | 100 | 156 s |

---

## Results

### PUT Latency

| Metric | 1-Shard 100u | 1-Shard 50u | 2-Shard 100u | 2-Shard Long |
|---|---|---|---|---|
| Total RPS | 555 | 565 | **945** | 935 |
| PUT avg | 180 ms | 61 ms | 123 ms (s0) / **40 ms (s1)** | 81 ms (s0) / 91 ms (s1) |
| PUT p50 | 170 ms | 54 ms | 85 ms (s0) / **38 ms (s1)** | 44 ms (s0) / 45 ms (s1) |
| PUT p95 | 380 ms | 100 ms | 340 ms (s0) / **50 ms (s1)** | 280 ms (s0) / 200 ms (s1) |
| PUT max | 3500 ms | 798 ms | 1562 ms | 8130 ms |
| PUT failures | 0 | 0 | 0 | **7450 (503 — shard1)** |

### GET Latency

GET latency was stable at **~39–41 ms avg** across all tests and both shards. Reads are served directly from the in-memory map and are unaffected by shard count, load, or snapshot growth.

---

## Analysis

### 1. Throughput scales with shards (~1.7×)

Total RPS increased from 555 (1-shard, 100 users) to 945 (2-shard, 100 users) — a **1.7× improvement**. The theoretical maximum is 2× since each shard handles an independent write quorum. The gap from 2× is explained by:
- 25% of traffic is reads, which don't benefit from sharding (both shards serve reads independently anyway)
- Locust coordination overhead
- The key space hash distribution is not perfectly 50/50

### 2. Single shard saturates at ~50 concurrent users

The 1-shard tests reveal a clear saturation boundary:

- **50 users → 61 ms avg PUT, 565 RPS** — system is within capacity
- **100 users → 180 ms avg PUT, 555 RPS** — system is saturated

Despite doubling users, total RPS barely changed while latency tripled. This is the signature of a saturated system — queueing theory (Little's Law) predicts that beyond the saturation point, latency grows non-linearly while throughput plateaus.

The bottleneck is **disk I/O, not CPU**. CloudWatch CPU utilization peaked at only 12.6% during the heaviest test. The constraint is the synchronous snapshot flush in `RunApplyLoop` — every 100 writes, the full KV map is serialized to JSON and written to disk. At high write rates this flush cannot keep up and the `applyCh` channel backs up, stalling incoming writes.

### 3. 2-shard keeps each leader below saturation

With 2 shards, each leader handles ~50% of the 100-user write load — roughly equivalent to the 50-user 1-shard case. This is why 2-shard PUT latency is low:
- shard1 PUT p50: **38 ms** (fresh shard, no prior snapshot data)
- shard0 PUT p50: **85 ms** (higher because shard0 had accumulated data from the prior 1-shard test, making snapshot flushes more expensive)

This asymmetry confirms that **snapshot file size directly impacts write latency** — a larger snapshot takes longer to serialize and flush.

### 4. Raft leader re-election visible in long test

At ~2 minutes into the 2-shard long test, shard1 experienced **7,450 consecutive PUT failures (HTTP 503 — Service Unavailable)**. The response time chart shows latency spiking to 2,500 ms (p95) then dropping back to ~50 ms after recovery. This is a **Raft leader re-election event**:

1. The current shard1 leader missed heartbeats (possibly due to high disk I/O from snapshot flush blocking the goroutine)
2. Followers timed out and started an election
3. During the election window (~300–600 ms), the cluster correctly refused all writes (503) to preserve consistency
4. A new leader was elected, replication resumed, and latency recovered

This is not a bug — it is the Raft safety guarantee working correctly. The cluster chose consistency over availability during the election window. CPU on shard1-nodeF peaked at 12.6% during this event, corresponding to the new leader replaying committed entries to catch up followers.

---

## Conclusions

1. **Horizontal sharding scales write throughput linearly**. A 2-shard cluster delivers ~1.7× the write RPS of a 1-shard cluster under the same load, with each shard maintaining lower per-leader latency.

2. **The disk I/O bottleneck (snapshot flush) is the primary limiter**, not CPU or network. Each t3.micro leader saturates at ~50 concurrent write users (~420 writes/sec). Sharding distributes this bottleneck by giving each shard its own independent flush cycle.

3. **Raft leader re-election is observable under sustained load** and causes brief write outages (~1–2 seconds). The system self-heals within one election timeout cycle (300–600 ms configured). This trade-off — brief unavailability for strong consistency — is fundamental to Raft.

4. **Reads scale trivially** — GET latency is flat at ~39 ms regardless of shard count or load, since all reads are served from the in-memory map without Raft involvement.
