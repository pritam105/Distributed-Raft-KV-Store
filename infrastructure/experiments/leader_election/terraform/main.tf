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

resource "aws_security_group" "exp" {
  name        = "${var.project}-sg"
  description = "Leader election experiment"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

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

locals {
  subnet_cidr = data.aws_subnet.first.cidr_block
  subnet_id   = data.aws_subnet.first.id

  node_ids = ["nodeA", "nodeB", "nodeC"]
  node_ips = [
    cidrhost(local.subnet_cidr, 10),
    cidrhost(local.subnet_cidr, 11),
    cidrhost(local.subnet_cidr, 12),
  ]

  peer_strings = [
    for i in range(3) : join(",", [
      for j in range(3) :
      "${local.node_ids[j]}@${local.node_ips[j]}:${var.app_port}"
      if j != i
    ])
  ]
}

resource "aws_instance" "node" {
  count                       = 3
  ami                         = data.aws_ami.al2023.id
  instance_type               = var.instance_type
  subnet_id                   = local.subnet_id
  private_ip                  = local.node_ips[count.index]
  vpc_security_group_ids      = [aws_security_group.exp.id]
  key_name                    = var.key_name
  associate_public_ip_address = true

  user_data = base64encode(templatefile("${path.module}/user_data_raft.sh.tpl", {
    node_id   = local.node_ids[count.index]
    peers     = local.peer_strings[count.index]
    app_port  = var.app_port
    image     = var.docker_image
    wal_path  = "/data/wal.log"
    snap_path = "/data/snapshot.json"
  }))

  tags = {
    Name       = "${var.project}-${local.node_ids[count.index]}"
    Experiment = "leader-election"
  }
}
