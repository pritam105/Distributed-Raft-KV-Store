#!/bin/bash
# Usage: ./restart_nodes.sh <path-to-pem>
# Pulls latest Docker image and restarts raft-node on all 6 nodes.

set -euo pipefail

PEM=~/Downloads/labsuser.pem
IMAGE="pritammane105/raft-kv:latest"

NODES=(
  "3.236.225.29"
  "100.54.101.62"
  "3.238.236.168"
  "100.53.242.242"
  "3.239.58.189"
  "3.235.173.220"
)

for ip in "${NODES[@]}"; do
  echo "=== Restarting $ip ==="
  ssh -i "$PEM" -o StrictHostKeyChecking=no ec2-user@"$ip" \
    "sudo docker pull $IMAGE && sudo systemctl restart raft-node"
  echo "=== Done $ip ==="
done

echo "All nodes restarted."
