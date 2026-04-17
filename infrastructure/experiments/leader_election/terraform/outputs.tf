output "public_ips" {
  value = {
    for i, inst in aws_instance.node :
    local.node_ids[i] => inst.public_ip
  }
}

output "status_urls" {
  description = "Check who is leader"
  value = {
    for i, inst in aws_instance.node :
    local.node_ids[i] => "http://${inst.public_ip}:${var.app_port}/status"
  }
}

output "ssh_commands" {
  value = {
    for i, inst in aws_instance.node :
    local.node_ids[i] => "ssh -i ${var.key_name}.pem ec2-user@${inst.public_ip}"
  }
}
