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

# ── Latest Amazon Linux 2023 AMI ──────────────────────────────────────────────

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

# ── VPC ───────────────────────────────────────────────────────────────────────

resource "aws_vpc" "raft" {
  cidr_block           = "10.10.0.0/16"
  enable_dns_hostnames = true
  tags = { Name = "${var.project}-vpc" }
}

resource "aws_internet_gateway" "igw" {
  vpc_id = aws_vpc.raft.id
  tags   = { Name = "${var.project}-igw" }
}

resource "aws_subnet" "node" {
  count                   = var.node_count
  vpc_id                  = aws_vpc.raft.id
  cidr_block              = cidrsubnet("10.10.0.0/16", 8, count.index)
  map_public_ip_on_launch = true
  tags = { Name = "${var.project}-subnet-${count.index}" }
}

resource "aws_route_table" "rt" {
  vpc_id = aws_vpc.raft.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.igw.id
  }
  tags = { Name = "${var.project}-rt" }
}

resource "aws_route_table_association" "rta" {
  count          = var.node_count
  subnet_id      = aws_subnet.node[count.index].id
  route_table_id = aws_route_table.rt.id
}

# ── Security Group ────────────────────────────────────────────────────────────

resource "aws_security_group" "raft" {
  name   = "${var.project}-sg"
  vpc_id = aws_vpc.raft.id

  # SSH
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Raft RPC + status + metrics
  ingress {
    from_port   = var.raft_port
    to_port     = var.raft_port
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
# nodeA=10.10.0.10  nodeB=10.10.1.10  nodeC=10.10.2.10
# Fixed so each node knows its peers at boot — no service discovery needed

locals {
  node_ids    = [for i in range(var.node_count) : "node${element(["A","B","C","D"], i)}"]
  private_ips = [for i in range(var.node_count) : "10.10.${i}.10"]

  peer_strings = [
    for i in range(var.node_count) : join(",", [
      for j in range(var.node_count) :
      "${local.node_ids[j]}@${local.private_ips[j]}:${var.raft_port}"
      if j != i
    ])
  ]
}

# ── EC2 Instances ─────────────────────────────────────────────────────────────

resource "aws_instance" "node" {
  count                  = var.node_count
  ami                    = data.aws_ami.al2023.id
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.node[count.index].id
  private_ip             = local.private_ips[count.index]
  vpc_security_group_ids = [aws_security_group.raft.id]
  key_name               = var.key_name

  user_data = base64encode(templatefile("${path.module}/user_data.sh", {
    node_id   = local.node_ids[count.index]
    peers     = local.peer_strings[count.index]
    raft_port = var.raft_port
  }))

  tags = {
    Name   = "${var.project}-${local.node_ids[count.index]}"
    NodeID = local.node_ids[count.index]
  }
}