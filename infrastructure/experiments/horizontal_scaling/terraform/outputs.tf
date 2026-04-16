output "shard0_public_ips" {
  value = {
    for i, inst in aws_instance.shard0 :
    local.shard0_ids[i] => inst.public_ip
  }
}

output "shard1_public_ips" {
  value = {
    for i, inst in aws_instance.shard1 :
    local.shard1_ids[i] => inst.public_ip
  }
}

output "status_urls" {
  description = "Check shard leaders — look for isLeader:true"
  value = merge(
    {
      for i, inst in aws_instance.shard0 :
      "shard0-${local.shard0_ids[i]}" => "http://${inst.public_ip}:${var.app_port}/status"
    },
    {
      for i, inst in aws_instance.shard1 :
      "shard1-${local.shard1_ids[i]}" => "http://${inst.public_ip}:${var.app_port}/status"
    }
  )
}

output "ssh_commands" {
  value = merge(
    {
      for i, inst in aws_instance.shard0 :
      "shard0-${local.shard0_ids[i]}" => "ssh -i ${var.key_name}.pem ec2-user@${inst.public_ip}"
    },
    {
      for i, inst in aws_instance.shard1 :
      "shard1-${local.shard1_ids[i]}" => "ssh -i ${var.key_name}.pem ec2-user@${inst.public_ip}"
    }
  )
}

output "locust_commands" {
  description = "Copy-paste after finding shard leaders via status_urls"
  value       = <<-EOT
    # 1-shard baseline (all traffic to shard0 leader):
    CLIENT_SHARDS_TOTAL=1 \
    CLIENT_SHARD_0_ADDRS=http://<shard0-leader-ip>:${var.app_port} \
    locust -f experiment.py ShardedUser \
      --host http://<shard0-leader-ip>:${var.app_port} \
      --users 100 --spawn-rate 10 --run-time 60s --headless --csv results_1shard

    # 2-shard (traffic split across both leaders):
    CLIENT_SHARDS_TOTAL=2 \
    CLIENT_SHARD_0_ADDRS=http://<shard0-leader-ip>:${var.app_port} \
    CLIENT_SHARD_1_ADDRS=http://<shard1-leader-ip>:${var.app_port} \
    locust -f experiment.py ShardedUser \
      --host http://<shard0-leader-ip>:${var.app_port} \
      --users 100 --spawn-rate 10 --run-time 60s --headless --csv results_2shard
  EOT
}
