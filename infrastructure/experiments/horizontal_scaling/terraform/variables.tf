variable "aws_region" {
  default = "us-east-1"
}

variable "instance_type" {
  default = "t3.micro"
}

variable "key_name" {
  description = "EC2 key pair name (must already exist in your AWS account)"
  type        = string
}

variable "project" {
  default = "raft-kv-hscale"
}

variable "app_port" {
  description = "Port used by both Raft RPC and KV HTTP API"
  default     = 8000
}

variable "docker_image" {
  description = "Docker image with node + simplekvs binaries — build and push before apply"
  type        = string
}
