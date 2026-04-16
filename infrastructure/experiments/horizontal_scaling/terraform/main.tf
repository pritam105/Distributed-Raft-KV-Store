terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

data "aws_ami" "al2023" {
  most_recent = true
  owners      = ["amazon"]
  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# ── Use default VPC (AWS Academy compatible) ──────────────────────────────────

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
  filter {
    name   = "defaultForAz"
    values = ["true"]
  }
}

data "aws_subnet" "first" {
  id = tolist(data.aws_subnets.default.ids)[0]
}

# ── Security Group ────────────────────────────────────────────────────────────

resource "aws_security_group" "exp" {
  name        = "${var.project}-sg"
  description = "Horizontal scaling experiment"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Locust (laptop) → nodes and node → node Raft RPC
  ingress {
    description = "KV API + Raft RPC"
    from_port   = var.app_port
    to_port     = var.app_port
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.project}-sg" }
}

# ── Fixed private IPs ─────────────────────────────────────────────────────────
# Shard 0: offsets 10-12   Shard 1: offsets 13-15

locals {
  subnet_cidr = data.aws_subnet.first.cidr_block
  subnet_id   = data.aws_subnet.first.id

  shard0_ids = ["nodeA", "nodeB", "nodeC"]
  shard0_ips = [
    cidrhost(local.subnet_cidr, 10),
    cidrhost(local.subnet_cidr, 11),
    cidrhost(local.subnet_cidr, 12),
  ]

  shard1_ids = ["nodeD", "nodeE", "nodeF"]
  shard1_ips = [
    cidrhost(local.subnet_cidr, 13),
    cidrhost(local.subnet_cidr, 14),
    cidrhost(local.subnet_cidr, 15),
  ]

  shard0_peers = [
    for i in range(3) : join(",", [
      for j in range(3) :
      "${local.shard0_ids[j]}@${local.shard0_ips[j]}:${var.app_port}"
      if j != i
    ])
  ]

  shard1_peers = [
    for i in range(3) : join(",", [
      for j in range(3) :
      "${local.shard1_ids[j]}@${local.shard1_ips[j]}:${var.app_port}"
      if j != i
    ])
  ]
}

# ── Shard 0 nodes ─────────────────────────────────────────────────────────────

resource "aws_instance" "shard0" {
  count                       = 3
  ami                         = data.aws_ami.al2023.id
  instance_type               = var.instance_type
  subnet_id                   = local.subnet_id
  private_ip                  = local.shard0_ips[count.index]
  vpc_security_group_ids      = [aws_security_group.exp.id]
  key_name                    = var.key_name
  associate_public_ip_address = true

  user_data = base64encode(templatefile("${path.module}/user_data_raft.sh.tpl", {
    node_id   = local.shard0_ids[count.index]
    peers     = local.shard0_peers[count.index]
    app_port  = var.app_port
    image     = var.docker_image
    wal_path  = "/data/wal.log"
    snap_path = "/data/snapshot.json"
  }))

  tags = {
    Name       = "${var.project}-shard0-${local.shard0_ids[count.index]}"
    Shard      = "0"
    Experiment = "horizontal-scaling"
  }
}

# ── Shard 1 nodes ─────────────────────────────────────────────────────────────

resource "aws_instance" "shard1" {
  count                       = 3
  ami                         = data.aws_ami.al2023.id
  instance_type               = var.instance_type
  subnet_id                   = local.subnet_id
  private_ip                  = local.shard1_ips[count.index]
  vpc_security_group_ids      = [aws_security_group.exp.id]
  key_name                    = var.key_name
  associate_public_ip_address = true

  user_data = base64encode(templatefile("${path.module}/user_data_raft.sh.tpl", {
    node_id   = local.shard1_ids[count.index]
    peers     = local.shard1_peers[count.index]
    app_port  = var.app_port
    image     = var.docker_image
    wal_path  = "/data/wal.log"
    snap_path = "/data/snapshot.json"
  }))

  tags = {
    Name       = "${var.project}-shard1-${local.shard1_ids[count.index]}"
    Shard      = "1"
    Experiment = "horizontal-scaling"
  }
}
