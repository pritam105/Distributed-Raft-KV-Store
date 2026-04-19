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
  alias  = "region_a"
  region = var.region_a
}

provider "aws" {
  alias  = "region_b"
  region = var.region_b
}

provider "aws" {
  alias  = "region_c"
  region = var.region_c
}

data "aws_ami" "al2023_a" {
  provider    = aws.region_a
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

data "aws_ami" "al2023_b" {
  provider    = aws.region_b
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

data "aws_ami" "al2023_c" {
  provider    = aws.region_c
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

resource "aws_vpc" "geo_a" {
  provider             = aws.region_a
  cidr_block           = "10.50.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${var.project}-vpc-a"
  }
}

resource "aws_vpc" "geo_b" {
  provider             = aws.region_b
  cidr_block           = "10.60.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${var.project}-vpc-b"
  }
}

resource "aws_vpc" "geo_c" {
  provider             = aws.region_c
  cidr_block           = "10.70.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${var.project}-vpc-c"
  }
}

resource "aws_internet_gateway" "geo_a" {
  provider = aws.region_a
  vpc_id   = aws_vpc.geo_a.id

  tags = {
    Name = "${var.project}-igw-a"
  }
}

resource "aws_internet_gateway" "geo_b" {
  provider = aws.region_b
  vpc_id   = aws_vpc.geo_b.id

  tags = {
    Name = "${var.project}-igw-b"
  }
}

resource "aws_internet_gateway" "geo_c" {
  provider = aws.region_c
  vpc_id   = aws_vpc.geo_c.id

  tags = {
    Name = "${var.project}-igw-c"
  }
}

resource "aws_subnet" "geo_a" {
  provider                = aws.region_a
  vpc_id                  = aws_vpc.geo_a.id
  cidr_block              = "10.50.1.0/24"
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.project}-public-subnet-a"
  }
}

resource "aws_subnet" "geo_b" {
  provider                = aws.region_b
  vpc_id                  = aws_vpc.geo_b.id
  cidr_block              = "10.60.1.0/24"
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.project}-public-subnet-b"
  }
}

resource "aws_subnet" "geo_c" {
  provider                = aws.region_c
  vpc_id                  = aws_vpc.geo_c.id
  cidr_block              = "10.70.1.0/24"
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.project}-public-subnet-c"
  }
}

resource "aws_route_table" "geo_a" {
  provider = aws.region_a
  vpc_id   = aws_vpc.geo_a.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.geo_a.id
  }

  tags = {
    Name = "${var.project}-rt-a"
  }
}

resource "aws_route_table" "geo_b" {
  provider = aws.region_b
  vpc_id   = aws_vpc.geo_b.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.geo_b.id
  }

  tags = {
    Name = "${var.project}-rt-b"
  }
}

resource "aws_route_table" "geo_c" {
  provider = aws.region_c
  vpc_id   = aws_vpc.geo_c.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.geo_c.id
  }

  tags = {
    Name = "${var.project}-rt-c"
  }
}

resource "aws_route_table_association" "geo_a" {
  provider       = aws.region_a
  subnet_id      = aws_subnet.geo_a.id
  route_table_id = aws_route_table.geo_a.id
}

resource "aws_route_table_association" "geo_b" {
  provider       = aws.region_b
  subnet_id      = aws_subnet.geo_b.id
  route_table_id = aws_route_table.geo_b.id
}

resource "aws_route_table_association" "geo_c" {
  provider       = aws.region_c
  subnet_id      = aws_subnet.geo_c.id
  route_table_id = aws_route_table.geo_c.id
}

resource "aws_security_group" "geo_a" {
  provider    = aws.region_a
  name        = "${var.project}-sg-a"
  description = "Geo-distributed Raft node A"
  vpc_id      = aws_vpc.geo_a.id

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
}

resource "aws_security_group" "geo_b" {
  provider    = aws.region_b
  name        = "${var.project}-sg-b"
  description = "Geo-distributed Raft node B"
  vpc_id      = aws_vpc.geo_b.id

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
}

resource "aws_security_group" "geo_c" {
  provider    = aws.region_c
  name        = "${var.project}-sg-c"
  description = "Geo-distributed Raft node C"
  vpc_id      = aws_vpc.geo_c.id

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
}

resource "tls_private_key" "generated" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "local_sensitive_file" "private_key" {
  filename        = "${path.module}/${var.project}.pem"
  content         = tls_private_key.generated.private_key_pem
  file_permission = "0600"
}

resource "aws_key_pair" "generated_a" {
  provider   = aws.region_a
  key_name   = "${var.project}-key"
  public_key = tls_private_key.generated.public_key_openssh
}

resource "aws_key_pair" "generated_b" {
  provider   = aws.region_b
  key_name   = "${var.project}-key"
  public_key = tls_private_key.generated.public_key_openssh
}

resource "aws_key_pair" "generated_c" {
  provider   = aws.region_c
  key_name   = "${var.project}-key"
  public_key = tls_private_key.generated.public_key_openssh
}

resource "aws_eip" "node_a" {
  provider = aws.region_a
  domain   = "vpc"
}

resource "aws_eip" "node_b" {
  provider = aws.region_b
  domain   = "vpc"
}

resource "aws_eip" "node_c" {
  provider = aws.region_c
  domain   = "vpc"
}

locals {
  node_a_id = "nodeA"
  node_b_id = "nodeB"
  node_c_id = "nodeC"

  node_a_peers = "${local.node_b_id}@${aws_eip.node_b.public_ip}:${var.app_port},${local.node_c_id}@${aws_eip.node_c.public_ip}:${var.app_port}"
  node_b_peers = "${local.node_a_id}@${aws_eip.node_a.public_ip}:${var.app_port},${local.node_c_id}@${aws_eip.node_c.public_ip}:${var.app_port}"
  node_c_peers = "${local.node_a_id}@${aws_eip.node_a.public_ip}:${var.app_port},${local.node_b_id}@${aws_eip.node_b.public_ip}:${var.app_port}"
}

resource "aws_instance" "node_a" {
  provider               = aws.region_a
  ami                    = data.aws_ami.al2023_a.id
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.geo_a.id
  vpc_security_group_ids = [aws_security_group.geo_a.id]
  key_name               = aws_key_pair.generated_a.key_name

  user_data = base64encode(templatefile("${path.module}/user_data_raft.sh.tpl", {
    node_id   = local.node_a_id
    peers     = local.node_a_peers
    app_port  = var.app_port
    image     = var.docker_image
    wal_path  = "/data/wal.log"
    snap_path = "/data/snapshot.json"
  }))

  tags = {
    Name       = "${var.project}-nodeA-${var.region_a}"
    Experiment = "geo-distributed"
    RegionRole = "A"
  }
}

resource "aws_instance" "node_b" {
  provider               = aws.region_b
  ami                    = data.aws_ami.al2023_b.id
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.geo_b.id
  vpc_security_group_ids = [aws_security_group.geo_b.id]
  key_name               = aws_key_pair.generated_b.key_name

  user_data = base64encode(templatefile("${path.module}/user_data_raft.sh.tpl", {
    node_id   = local.node_b_id
    peers     = local.node_b_peers
    app_port  = var.app_port
    image     = var.docker_image
    wal_path  = "/data/wal.log"
    snap_path = "/data/snapshot.json"
  }))

  tags = {
    Name       = "${var.project}-nodeB-${var.region_b}"
    Experiment = "geo-distributed"
    RegionRole = "B"
  }
}

resource "aws_instance" "node_c" {
  provider               = aws.region_c
  ami                    = data.aws_ami.al2023_c.id
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.geo_c.id
  vpc_security_group_ids = [aws_security_group.geo_c.id]
  key_name               = aws_key_pair.generated_c.key_name

  user_data = base64encode(templatefile("${path.module}/user_data_raft.sh.tpl", {
    node_id   = local.node_c_id
    peers     = local.node_c_peers
    app_port  = var.app_port
    image     = var.docker_image
    wal_path  = "/data/wal.log"
    snap_path = "/data/snapshot.json"
  }))

  tags = {
    Name       = "${var.project}-nodeC-${var.region_c}"
    Experiment = "geo-distributed"
    RegionRole = "C"
  }
}

resource "aws_eip_association" "node_a" {
  provider      = aws.region_a
  instance_id   = aws_instance.node_a.id
  allocation_id = aws_eip.node_a.id
}

resource "aws_eip_association" "node_b" {
  provider      = aws.region_b
  instance_id   = aws_instance.node_b.id
  allocation_id = aws_eip.node_b.id
}

resource "aws_eip_association" "node_c" {
  provider      = aws.region_c
  instance_id   = aws_instance.node_c.id
  allocation_id = aws_eip.node_c.id
}
