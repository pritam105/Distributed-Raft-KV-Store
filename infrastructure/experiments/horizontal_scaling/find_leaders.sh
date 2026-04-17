#!/bin/bash
# Polls all 6 nodes and prints the leader for each shard.

SHARD0=(
  "nodeA|3.236.225.29"
  "nodeB|100.54.101.62"
  "nodeC|3.238.236.168"
)
SHARD1=(
  "nodeD|100.53.242.242"
  "nodeE|3.239.58.189"
  "nodeF|3.235.173.220"
)

find_leader() {
  local shard=$1
  shift
  local nodes=("$@")
  echo "--- Shard $shard ---"
  for entry in "${nodes[@]}"; do
    local id="${entry%%|*}"
    local ip="${entry##*|}"
    local resp
    resp=$(curl -sf --max-time 3 "http://$ip:8000/status" 2>/dev/null)
    if [ -z "$resp" ]; then
      echo "  $id ($ip): unreachable"
    else
      local leader
      leader=$(echo "$resp" | grep -o '"isLeader":[a-z]*' | cut -d: -f2)
      local term
      term=$(echo "$resp" | grep -o '"term":[0-9]*' | cut -d: -f2)
      echo "  $id ($ip): isLeader=$leader term=$term"
    fi
  done
}

find_leader 0 "${SHARD0[@]}"
find_leader 1 "${SHARD1[@]}"
