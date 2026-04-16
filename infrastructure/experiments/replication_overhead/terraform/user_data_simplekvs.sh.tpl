#!/bin/bash
set -euo pipefail
exec > /var/log/simplekvs-bootstrap.log 2>&1

echo "=== bootstrapping simplekvs ==="

dnf update -y
dnf install -y docker
systemctl start docker
systemctl enable docker

docker pull ${image}

mkdir -p /data

cat > /etc/systemd/system/simplekvs.service << EOF
[Unit]
Description=SimpleKVS (single-node baseline)
After=docker.service
Requires=docker.service

[Service]
ExecStart=/usr/bin/docker run --rm \
  --name simplekvs \
  -p ${app_port}:8080 \
  -v /data:/data \
  -e SERVICE_TYPE=simplekvs \
  -e KVS_ADDR=0.0.0.0:8080 \
  -e KVS_WAL_PATH=/data/wal.log \
  -e KVS_SNAPSHOT_PATH=/data/snapshot.json \
  ${image}
ExecStop=/usr/bin/docker stop simplekvs
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable simplekvs
systemctl start simplekvs

echo "=== simplekvs ready ==="
