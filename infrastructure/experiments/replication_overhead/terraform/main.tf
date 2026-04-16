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

# Pick the first available default subnet — all nodes in same AZ
# to minimise inter-node latency variance during experiments
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
  description = "Replication overhead experiment"
  vpc_id      = data.aws_vpc.default.id

  # SSH — for debugging
  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # App port — Locust (your laptop) → EC2 and EC2 → EC2 Raft RPC
  ingress {
    description = "KV API + Raft RPC"
    from_port   = var.app_port
    to_port     = var.app_port
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # SimpleKVS runs on 8080 internally, mapped to app_port externally
  ingress {
    description = "SimpleKVS internal port"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.default.cidr_block]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.project}-sg" }
}

# ── Fixed private IPs within the first default subnet ────────────────────────
# cidrhost picks a deterministic IP so nodes know peers before boot

locals {
  subnet_cidr = data.aws_subnet.first.cidr_block
  subnet_id   = data.aws_subnet.first.id

  raft_ids = ["nodeA", "nodeB", "nodeC"]
  raft_ips = [
    cidrhost(local.subnet_cidr, 10),
    cidrhost(local.subnet_cidr, 11),
    cidrhost(local.subnet_cidr, 12),
  ]

  raft_peers = [
    for i in range(3) : join(",", [
      for j in range(3) :
      "${local.raft_ids[j]}@${local.raft_ips[j]}:${var.app_port}"
      if j != i
    ])
  ]

  simplekvs_ip = cidrhost(local.subnet_cidr, 13)
}

# ── Raft nodes ────────────────────────────────────────────────────────────────

resource "aws_instance" "raft" {
  count                       = 3
  ami                         = data.aws_ami.al2023.id
  instance_type               = var.instance_type
  subnet_id                   = local.subnet_id
  private_ip                  = local.raft_ips[count.index]
  vpc_security_group_ids      = [aws_security_group.exp.id]
  key_name                    = var.key_name
  associate_public_ip_address = true

  user_data = base64encode(templatefile("${path.module}/user_data_raft.sh.tpl", {
    node_id   = local.raft_ids[count.index]
    peers     = local.raft_peers[count.index]
    app_port  = var.app_port
    image     = var.docker_image
    wal_path  = "/data/wal.log"
    snap_path = "/data/snapshot.json"
  }))

  tags = {
    Name       = "${var.project}-${local.raft_ids[count.index]}"
    Experiment = "replication-overhead"
  }
}

# ── SimpleKVS node ────────────────────────────────────────────────────────────

resource "aws_instance" "simplekvs" {
  ami                         = data.aws_ami.al2023.id
  instance_type               = var.instance_type
  subnet_id                   = local.subnet_id
  private_ip                  = local.simplekvs_ip
  vpc_security_group_ids      = [aws_security_group.exp.id]
  key_name                    = var.key_name
  associate_public_ip_address = true

  user_data = base64encode(templatefile("${path.module}/user_data_simplekvs.sh.tpl", {
    app_port = var.app_port
    image    = var.docker_image
  }))

  tags = {
    Name       = "${var.project}-simplekvs"
    Experiment = "replication-overhead"
  }
}
