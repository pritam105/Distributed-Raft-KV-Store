#!/bin/bash
set -euo pipefail
exec > /var/log/raft-bootstrap.log 2>&1

echo "=== bootstrapping ${node_id} ==="

# Install Docker
dnf update -y
dnf install -y docker
systemctl start docker
systemctl enable docker

# Pull image
docker pull kevinjohnson29/raft-node:latest

# Create systemd service
cat > /etc/systemd/system/raft-node.service << EOF
[Unit]
Description=Raft node ${node_id}
After=docker.service
Requires=docker.service

[Service]
ExecStart=/usr/bin/docker run --rm \
  --name raft-node \
  -p ${raft_port}:${raft_port} \
  -e RAFT_NODE_ID=${node_id} \
  -e RAFT_PEERS=${peers} \
  -e RAFT_ADDR=0.0.0.0:${raft_port} \
  kevinjohnson29/raft-node:latest
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