terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
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

resource "aws_vpc" "geo" {
  cidr_block           = "10.40.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${var.project}-vpc"
  }
}

resource "aws_internet_gateway" "geo" {
  vpc_id = aws_vpc.geo.id

  tags = {
    Name = "${var.project}-igw"
  }
}

resource "aws_subnet" "geo" {
  vpc_id                  = aws_vpc.geo.id
  cidr_block              = "10.40.1.0/24"
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.project}-public-subnet"
  }
}

resource "aws_route_table" "geo" {
  vpc_id = aws_vpc.geo.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.geo.id
  }

  tags = {
    Name = "${var.project}-rt"
  }
}

resource "aws_route_table_association" "geo" {
  subnet_id      = aws_subnet.geo.id
  route_table_id = aws_route_table.geo.id
}

resource "aws_security_group" "geo" {
  name        = "${var.project}-sg"
  description = "Co-located geo tradeoff experiment"
  vpc_id      = aws_vpc.geo.id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "KV API and Raft RPC"
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

resource "tls_private_key" "generated" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "generated" {
  key_name   = "${var.project}-key"
  public_key = tls_private_key.generated.public_key_openssh
}

resource "local_sensitive_file" "private_key" {
  filename        = "${path.module}/${var.project}.pem"
  content         = tls_private_key.generated.private_key_pem
  file_permission = "0600"
}

locals {
  subnet_cidr = aws_subnet.geo.cidr_block
  subnet_id   = aws_subnet.geo.id

  node_ids = ["nodeA", "nodeB", "nodeC"]
  node_ips = [
    cidrhost(local.subnet_cidr, 40),
    cidrhost(local.subnet_cidr, 41),
    cidrhost(local.subnet_cidr, 42),
  ]

  peers = [
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
  vpc_security_group_ids      = [aws_security_group.geo.id]
  key_name                    = aws_key_pair.generated.key_name
  associate_public_ip_address = true

  user_data = base64encode(templatefile("${path.module}/user_data_raft.sh.tpl", {
    node_id   = local.node_ids[count.index]
    peers     = local.peers[count.index]
    app_port  = var.app_port
    image     = var.docker_image
    wal_path  = "/data/wal.log"
    snap_path = "/data/snapshot.json"
  }))

  tags = {
    Name       = "${var.project}-${local.node_ids[count.index]}"
    Experiment = "geo-colocated"
  }
}
