variable "aws_region" {
  default = "us-west-2"
}

variable "node_count" {
  description = "Number of Raft nodes — 3 or 4"
  type        = number
  default     = 3
}

variable "instance_type" {
  default = "t3.micro"
}

variable "key_name" {
  description = "EC2 key pair name (must already exist in your AWS account)"
  type        = string
}

variable "project" {
  default = "raft-kv"
}

variable "raft_port" {
  default = 7000
}