# Geo-Distribution Tradeoff Experiment

This experiment compares a co-located Raft shard against a geo-distributed Raft shard. The goal is to understand how replica placement affects write latency, read latency, and replica convergence behavior.

## What We Tested

We ran the same workload against two 3-node Raft deployments:

- **Co-located deployment**: all three Raft nodes run in the same AWS region/VPC.
- **Geo-distributed deployment**: the three Raft nodes run across multiple AWS regions.

Each deployment forms one Raft shard with one leader and two followers. The experiment script repeatedly finds the current leader, writes a unique key through the leader, immediately reads the same key from all replicas, and then polls until every replica returns the latest value.

This is meant to measure the tradeoff between:

- faster consensus when replicas are close together
- potentially better read locality when replicas are spread across regions
- slower convergence when followers are farther from the leader

## Metrics Collected

The script collects the following metrics for each round:

- `write_latency_ms`: time for a `PUT` request to the Raft leader to complete
- `read_latency_avg_ms`: average immediate `GET` latency across all replicas
- `read_latency_min_ms`: fastest immediate replica read
- `read_latency_max_ms`: slowest immediate replica read
- `immediate_stale_reads`: number of replicas that did not immediately return the newly written value
- `convergence_ms`: time until all replicas returned the latest value

The CSV output is written by each experiment run, and the comparison script summarizes both CSVs.

## How To Re-run

Build and push the experiment Docker image from the repository root:

```bash
docker buildx build --no-cache --platform linux/amd64 \
  -f infrastructure/experiments/Dockerfile \
  -t shreyansmulkutkar/raft-kv:geo-v2 \
  --push .
```

Deploy the co-located experiment:

```bash
cd infrastructure/experiments/geo_colocated/terraform
terraform init
terraform apply -var='docker_image=shreyansmulkutkar/raft-kv:geo-v2'
terraform output node_urls
terraform output experiment_command
```

Run the co-located experiment from `infrastructure/experiments/geo_colocated`:

```bash
python3 experiment.py \
  --nodes http://<nodeA-ip>:8000 http://<nodeB-ip>:8000 http://<nodeC-ip>:8000 \
  --rounds 100 \
  --csv terraform/colocated_results.csv
```

Deploy the geo-distributed experiment:

```bash
cd infrastructure/experiments/geo_distributed/terraform
terraform init
terraform apply \
  -var='docker_image=shreyansmulkutkar/raft-kv:geo-v2' \
  -var='region_a=us-east-1' \
  -var='region_b=us-east-2' \
  -var='region_c=us-west-2'
terraform output node_urls
terraform output experiment_command
```

Run the geo-distributed experiment from `infrastructure/experiments/geo_distributed`:

```bash
python3 experiment.py \
  --nodes http://<nodeA-ip>:8000 http://<nodeB-ip>:8000 http://<nodeC-ip>:8000 \
  --rounds 100 \
  --csv terraform/geo_distributed_results.csv
```

Compare the results from `infrastructure/experiments`:

```bash
python3 compare_geo_results.py \
  --colocated geo_colocated/terraform/colocated_results.csv \
  --distributed geo_distributed/terraform/geo_distributed_results.csv \
  --output geo_tradeoff_report.md
```

## Results

Both experiments completed 100 successful rounds.

| Deployment | Avg Write | P50 Write | P95 Write | Avg Read | P50 Read | P95 Read | Avg Convergence | P50 Convergence | P95 Convergence | Immediate Stale Reads |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Co-located | 266.26 ms | 196.53 ms | 684.26 ms | 251.60 ms | 244.15 ms | 496.70 ms | 799.20 ms | 783.24 ms | 1546.52 ms | 0 / 300 |
| Geo-distributed | 311.34 ms | 304.17 ms | 686.52 ms | 363.05 ms | 371.86 ms | 570.97 ms | 1090.10 ms | 1147.59 ms | 1738.67 ms | 0 / 300 |

The generated comparison report showed:

- Geo-distributed average write latency was **1.17x** the co-located write latency.
- Geo-distributed average read latency was **1.44x** the co-located read latency.
- Geo-distributed average convergence time was **1.36x** the co-located convergence time.
- No immediate stale reads were observed in either run.

## Interpretation

The geo-distributed deployment had higher average write latency. This is expected because each write must be replicated through Raft and acknowledged by a majority. When replicas are spread across regions, the leader needs to communicate over longer network paths before a write can be committed.

Read latency was also higher in the geo-distributed run. In this experiment, reads were issued from the same client location to all replicas, so spreading replicas across regions did not provide a client-proximity benefit. Instead, the client had to contact farther-away replicas, increasing average read latency.

Convergence time was higher for the geo-distributed deployment. This suggests that even after the leader accepts a write, followers in other regions take longer to expose the latest value. This matches the expected tradeoff: geographic placement can improve locality for nearby clients, but it makes cross-region coordination slower.

The stale-read count was zero in both experiments. This means that in the sampled rounds, all replicas returned the latest value immediately after the write completed. This may be because the current Raft implementation applies committed entries quickly enough relative to the measurement interval. It does not mean stale reads are impossible; it only means this run did not observe them.

## Takeaways

The experiment supports the expected geo-distribution tradeoff:

- Co-located replicas provide lower write latency and faster convergence.
- Geo-distributed replicas increase coordination cost.
- Geo-distribution did not improve read latency in this setup because the client was not placed near each remote region.
- To demonstrate the read-locality benefit more clearly, future runs should place clients in multiple regions and compare each client's nearest follower against the leader region.

## Limitations

This experiment is useful as a first comparison, but it has some limitations:

- The client was not deployed in multiple regions, so the read-locality benefit of geo-distribution was not fully tested.
- The script measures immediate follower visibility from one client machine, not from clients near each follower.
- The current system uses a simplified Raft implementation and does not yet include production-grade log persistence or optimized follower-read semantics.
- The experiment used 100 sequential rounds, not a high-concurrency Locust workload.

Future versions can improve this by running regional clients, separating leader reads from follower reads, and measuring throughput under concurrent load.
