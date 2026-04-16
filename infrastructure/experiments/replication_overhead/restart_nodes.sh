#!/bin/bash
# Usage: ./restart_nodes.sh <path-to-pem>
# Pulls latest Docker image and restarts raft-node on all EC2 instances.

set -euo pipefail

PEM=${1:?"Usage: $0 <path-to-pem>"}
IMAGE="pritammane105/raft-kv:latest"

NODES=(
  "100.48.44.193"
  "13.219.179.116"
  "18.205.41.121"
)

for ip in "${NODES[@]}"; do
  echo "=== Restarting $ip ==="
  ssh -i "$PEM" -o StrictHostKeyChecking=no ec2-user@"$ip" \
    "sudo docker pull $IMAGE && sudo systemctl restart raft-node"
  echo "=== Done $ip ==="
done

echo "All nodes restarted."
