variable "aws_region" {
  default = "us-east-1"
}

variable "instance_type" {
  default = "t3.micro"
}

variable "project" {
  default = "raft-kv-geo-colocated"
}

variable "app_port" {
  description = "Port used by both Raft RPC and KV HTTP API"
  default     = 8000
}

variable "docker_image" {
  description = "Docker image with node and simplekvs binaries"
  type        = string
}
