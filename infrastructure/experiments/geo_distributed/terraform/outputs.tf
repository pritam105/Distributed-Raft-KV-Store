output "node_public_ips" {
  value = {
    nodeA = aws_eip.node_a.public_ip
    nodeB = aws_eip.node_b.public_ip
    nodeC = aws_eip.node_c.public_ip
  }
}

output "node_regions" {
  value = {
    nodeA = var.region_a
    nodeB = var.region_b
    nodeC = var.region_c
  }
}

output "node_urls" {
  value = [
    "http://${aws_eip.node_a.public_ip}:${var.app_port}",
    "http://${aws_eip.node_b.public_ip}:${var.app_port}",
    "http://${aws_eip.node_c.public_ip}:${var.app_port}",
  ]
}

output "status_urls" {
  value = {
    nodeA = "http://${aws_eip.node_a.public_ip}:${var.app_port}/status"
    nodeB = "http://${aws_eip.node_b.public_ip}:${var.app_port}/status"
    nodeC = "http://${aws_eip.node_c.public_ip}:${var.app_port}/status"
  }
}

output "experiment_command" {
  value = "python3 ../experiment.py --nodes http://${aws_eip.node_a.public_ip}:${var.app_port} http://${aws_eip.node_b.public_ip}:${var.app_port} http://${aws_eip.node_c.public_ip}:${var.app_port} --rounds 100 --csv geo_distributed_results.csv"
}

output "private_key_path" {
  value = local_sensitive_file.private_key.filename
}

output "reset_command" {
  value = "../reset_nodes.sh ${local_sensitive_file.private_key.filename} ${aws_eip.node_a.public_ip} ${aws_eip.node_b.public_ip} ${aws_eip.node_c.public_ip}"
}
