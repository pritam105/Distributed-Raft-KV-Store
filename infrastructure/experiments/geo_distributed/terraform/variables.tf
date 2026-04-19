variable "region_a" {
  default = "us-east-1"
}

variable "region_b" {
  default = "us-west-2"
}

variable "region_c" {
  default = "eu-west-1"
}

variable "instance_type" {
  default = "t3.micro"
}

variable "project" {
  default = "raft-kv-geo-distributed"
}

variable "app_port" {
  description = "Port used by both Raft RPC and KV HTTP API"
  default     = 8000
}

variable "docker_image" {
  description = "Docker image with node and simplekvs binaries"
  type        = string
}
