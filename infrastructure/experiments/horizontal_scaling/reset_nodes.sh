#!/bin/bash
# Usage: ./reset_nodes.sh <path-to-pem>
# Stops raft-node, clears WAL + snapshot, restarts clean.
# Wait ~5s after running before starting a new Locust test.

set -euo pipefail

PEM=~/Downloads/labsuser.pem

NODES=(
  "3.236.225.29"
  "100.54.101.62"
  "3.238.236.168"
  "100.53.242.242"
  "3.239.58.189"
  "3.235.173.220"
)

for ip in "${NODES[@]}"; do
  echo "=== Resetting $ip ==="
  ssh -i "$PEM" -o StrictHostKeyChecking=no ec2-user@"$ip" "
    sudo systemctl stop raft-node || true
    sudo rm -f /data/wal.log /data/snapshot.json
    sudo systemctl start raft-node
  "
  echo "=== Done $ip ==="
done

echo ""
echo "All nodes reset. Wait ~5 seconds for Raft elections before running Locust."
