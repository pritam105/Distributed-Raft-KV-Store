output "node_public_ips" {
  value = {
    for i, inst in aws_instance.node :
    local.node_ids[i] => inst.public_ip
  }
}

output "node_urls" {
  value = [
    for inst in aws_instance.node :
    "http://${inst.public_ip}:${var.app_port}"
  ]
}

output "status_urls" {
  value = {
    for i, inst in aws_instance.node :
    local.node_ids[i] => "http://${inst.public_ip}:${var.app_port}/status"
  }
}

output "experiment_command" {
  value = "python3 ../experiment.py --nodes ${join(" ", [for inst in aws_instance.node : "http://${inst.public_ip}:${var.app_port}"])} --rounds 100 --csv colocated_results.csv"
}

output "private_key_path" {
  value = local_sensitive_file.private_key.filename
}

output "reset_command" {
  value = "../reset_nodes.sh ${local_sensitive_file.private_key.filename} ${join(" ", [for inst in aws_instance.node : inst.public_ip])}"
}
