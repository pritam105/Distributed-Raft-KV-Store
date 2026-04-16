output "raft_public_ips" {
  value = {
    for i, inst in aws_instance.raft :
    local.raft_ids[i] => inst.public_ip
  }
}

output "simplekvs_public_ip" {
  value = aws_instance.simplekvs.public_ip
}

output "status_urls" {
  description = "Check Raft leader — look for isLeader:true"
  value = {
    for i, inst in aws_instance.raft :
    local.raft_ids[i] => "http://${inst.public_ip}:${var.app_port}/status"
  }
}

output "ssh_commands" {
  value = merge(
    {
      for i, inst in aws_instance.raft :
      local.raft_ids[i] => "ssh -i ${var.key_name}.pem ec2-user@${inst.public_ip}"
    },
    {
      simplekvs = "ssh -i ${var.key_name}.pem ec2-user@${aws_instance.simplekvs.public_ip}"
    }
  )
}

output "locust_commands" {
  description = "Copy-paste these to run the experiment from your laptop"
  value = <<-EOT
    # 1. Baseline — SingleNode SimpleKVS
    locust -f experiment.py SimpleKVSUser \
      --host http://${aws_instance.simplekvs.public_ip}:${var.app_port} \
      --users 50 --spawn-rate 5 --run-time 60s --headless --csv results_simplekvs

    # 2. Find Raft leader first, then:
    locust -f experiment.py RaftUser \
      --host http://<raft-leader-public-ip>:${var.app_port} \
      --users 50 --spawn-rate 5 --run-time 60s --headless --csv results_raft
  EOT
}
