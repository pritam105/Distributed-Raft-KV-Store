#!/bin/bash
set -euo pipefail
exec > /var/log/raft-bootstrap.log 2>&1

echo "=== bootstrapping ${node_id} ==="

# Install Go
dnf update -y
dnf install -y golang git

# Clone repo and build
cd /home/ec2-user
git clone https://github.com/pritam105/Distributed-Raft-KV-Store.git repo
cd repo
export HOME=/root
export GOPATH=/root/go
export GOMODCACHE=/root/go/pkg/mod
go build -o /usr/local/bin/raft-node ./cmd/node

# Create systemd service
cat > /etc/systemd/system/raft-node.service << EOF
[Unit]
Description=Raft node ${node_id}
After=network.target

[Service]
ExecStart=/usr/local/bin/raft-node
Restart=on-failure
RestartSec=3
Environment="RAFT_NODE_ID=${node_id}"
Environment="RAFT_PEERS=${peers}"
Environment="RAFT_ADDR=0.0.0.0:${raft_port}"
StandardOutput=append:/var/log/raft-node.log
StandardError=append:/var/log/raft-node.log

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable raft-node
systemctl start raft-node

echo "=== ${node_id} ready ==="