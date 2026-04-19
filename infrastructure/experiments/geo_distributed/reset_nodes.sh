#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "usage: $0 <ssh-key.pem> <node-ip> [node-ip ...]" >&2
  exit 1
fi

KEY="$1"
shift

for ip in "$@"; do
  echo "resetting $ip"
  ssh -o StrictHostKeyChecking=no -i "$KEY" "ec2-user@$ip" \
    "sudo systemctl stop raft-node || true; sudo rm -rf /data/*; sudo systemctl start raft-node"
done

echo "done"
