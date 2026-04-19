# Geo Tradeoff Experiment: Geo-distributed Raft Group

This experiment deploys one 3-node Raft shard across three AWS regions. It is intended to be compared against `../geo_colocated`.

What it measures:

- Write latency to the current Raft leader
- Read latency from each replica
- Immediate stale follower-read rate
- Time for a newly written value to become visible on every replica

This setup uses Elastic IPs so Raft peer addresses are known before instance boot and remain stable across restarts.

## Deploy

```bash
cd terraform
terraform init
terraform apply \
  -var='docker_image=<your-image>'
```

Terraform generates one SSH key and registers it as an EC2 key pair in each configured region. The private key is written to `terraform/raft-kv-geo-distributed.pem`.

Find node URLs:

```bash
terraform output node_urls
terraform output status_urls
terraform output private_key_path
```

## Run

From this directory:

```bash
python3 experiment.py \
  --nodes http://<nodeA-ip>:8000 http://<nodeB-ip>:8000 http://<nodeC-ip>:8000 \
  --rounds 100 \
  --csv geo_distributed_results.csv
```

Compare the output against the co-located experiment. The expected trend is higher write latency because Raft consensus crosses regions, but potentially lower read latency for clients near a follower if follower reads are used.
