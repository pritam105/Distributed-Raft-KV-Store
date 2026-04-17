#!/bin/bash
set -euo pipefail
exec > /var/log/raft-bootstrap.log 2>&1

echo "=== bootstrapping Raft node ${node_id} ==="

dnf update -y
dnf install -y docker
systemctl start docker
systemctl enable docker

docker pull ${image}

mkdir -p /data

cat > /etc/systemd/system/raft-node.service << EOF
[Unit]
Description=Raft KV node ${node_id}
After=docker.service
Requires=docker.service

[Service]
ExecStart=/usr/bin/docker run --rm \
  --name raft-node \
  -p ${app_port}:${app_port} \
  -v /data:/data \
  -e SERVICE_TYPE=node \
  -e RAFT_NODE_ID=${node_id} \
  -e RAFT_PEERS=${peers} \
  -e RAFT_ADDR=0.0.0.0:${app_port} \
  -e RAFT_WAL_PATH=${wal_path} \
  -e RAFT_SNAPSHOT_PATH=${snap_path} \
  ${image}
ExecStop=/usr/bin/docker stop raft-node
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable raft-node
systemctl start raft-node

echo "=== ${node_id} ready ==="
