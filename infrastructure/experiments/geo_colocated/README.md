# Geo Tradeoff Experiment: Co-located Raft Group

This experiment deploys one 3-node Raft shard in a single AWS region and availability zone. It is the baseline for the geo-distribution tradeoff experiment.

What it measures:

- Write latency to the current Raft leader
- Read latency from each replica
- Immediate stale follower-read rate
- Time for a newly written value to become visible on every replica

The same measurement script is also used by `../geo_distributed` so the results are comparable.

## Deploy

```bash
cd terraform
terraform init
terraform apply \
  -var='docker_image=<your-image>'
```

Terraform generates an EC2 key pair and writes the private key to `terraform/raft-kv-geo-colocated.pem`.

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
  --csv colocated_results.csv
```

The script automatically finds the leader before each write.
