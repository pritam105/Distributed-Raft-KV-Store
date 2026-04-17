"""
Horizontal Scaling Experiment — Locust load test
Compares 1-shard (3 nodes) vs 2-shard (6 nodes) setup.

The router hashes each key and sends to the correct shard.

Install:  pip install locust requests
Run against 1-shard (point both shards at same cluster):
  CLIENT_SHARDS_TOTAL=1 \
  CLIENT_SHARD_0_ADDRS=http://<shard0-leader>:8000 \
  locust -f experiment.py ShardedUser \
    --host http://<shard0-leader>:8000 \
    --users 100 --spawn-rate 10 --run-time 60s --headless

Run against 2-shard:
  CLIENT_SHARDS_TOTAL=2 \
  CLIENT_SHARD_0_ADDRS=http://<shard0-leader>:8000 \
  CLIENT_SHARD_1_ADDRS=http://<shard1-leader>:8000 \
  locust -f experiment.py ShardedUser \
    --host http://<shard0-leader>:8000 \
    --users 100 --spawn-rate 10 --run-time 60s --headless
"""

import os
import random
import string

from locust import HttpUser, between, task

# Shared set of keys confirmed written — reads only query keys that exist.
# Locust uses gevent (cooperative multitasking) so a plain list is safe.
WRITTEN_KEYS: list[str] = []
MAX_TRACKED_KEYS = 5000


def fnv1a_32(s: str) -> int:
    """FNV-1a 32-bit — matches Go's fnv.New32a() used in shard/hashing.go."""
    FNV_OFFSET = 0x811C9DC5
    FNV_PRIME  = 0x01000193
    h = FNV_OFFSET
    for b in s.encode():
        h ^= b
        h = (h * FNV_PRIME) & 0xFFFFFFFF
    return h


def key_to_shard(key: str, total_shards: int) -> int:
    return fnv1a_32(key) % total_shards


def random_value(length=8):
    return "".join(random.choices(string.ascii_lowercase, k=length))


def build_shard_map():
    """Read CLIENT_SHARD_N_ADDRS env vars and return {shard_id: [addr, ...]}."""
    total = int(os.environ.get("CLIENT_SHARDS_TOTAL", "1"))
    shards = {}
    for i in range(total):
        addrs = os.environ.get(f"CLIENT_SHARD_{i}_ADDRS", "").split(",")
        shards[i] = [a.strip() for a in addrs if a.strip()]
    return total, shards


TOTAL_SHARDS, SHARD_MAP = build_shard_map()


class ShardedUser(HttpUser):
    wait_time = between(0.01, 0.05)

    @task(3)
    def write(self):
        key = f"key-{random.randint(0, 9999)}"
        shard_id = key_to_shard(key, TOTAL_SHARDS)
        addrs = SHARD_MAP.get(shard_id, SHARD_MAP[0])
        target = random.choice(addrs)
        resp = self.client.put(
            f"{target}/v1/keys/{key}",
            json={"value": random_value()},
            name=f"/v1/keys/[key] PUT shard{shard_id}",
        )
        if resp and resp.status_code == 200:
            if len(WRITTEN_KEYS) < MAX_TRACKED_KEYS:
                WRITTEN_KEYS.append(key)
            else:
                WRITTEN_KEYS[random.randint(0, MAX_TRACKED_KEYS - 1)] = key

    @task(1)
    def read(self):
        if not WRITTEN_KEYS:
            return  # skip until at least one key has been written
        key = random.choice(WRITTEN_KEYS)
        shard_id = key_to_shard(key, TOTAL_SHARDS)
        addrs = SHARD_MAP.get(shard_id, SHARD_MAP[0])
        target = random.choice(addrs)
        self.client.get(
            f"{target}/v1/keys/{key}",
            name=f"/v1/keys/[key] GET shard{shard_id}",
        )
