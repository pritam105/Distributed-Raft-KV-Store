#!/bin/bash
# Usage: ./reset_nodes.sh <path-to-pem>
# Stops raft-node, clears WAL + snapshot, restarts clean.
# Wait ~5s after running before starting a new Locust test.

set -euo pipefail

PEM=${1:?"Usage: $0 <path-to-pem>"}

RAFT_NODES=(
  "100.48.44.193"
  "13.219.179.116"
  "18.205.41.121"
)
SIMPLEKVS="98.93.5.127"

reset_node() {
  local ip=$1
  local service=$2
  echo "=== Resetting $ip ==="
  ssh -i "$PEM" -o StrictHostKeyChecking=no ec2-user@"$ip" "
    sudo systemctl stop $service || true
    sudo rm -f /data/wal.log /data/snapshot.json
    sudo systemctl start $service
  "
  echo "=== Done $ip ==="
}

for ip in "${RAFT_NODES[@]}"; do
  reset_node "$ip" "raft-node"
done

reset_node "$SIMPLEKVS" "simplekvs"

echo ""
echo "All nodes reset. Wait ~5 seconds for Raft election before running Locust."
