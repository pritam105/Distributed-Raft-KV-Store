"""
Replication Overhead Experiment — Locust load test
Compares single-node SimpleKVS vs 3-node Raft shard.

Install:  pip install locust requests
Run:
  # Against SimpleKVS (baseline)
  locust -f experiment.py SimpleKVSUser \
    --host http://<simplekvs-ip>:8000 \
    --users 50 --spawn-rate 5 --run-time 60s --headless

  # Against Raft cluster (find leader first)
  locust -f experiment.py RaftUser \
    --host http://<leader-ip>:8000 \
    --users 50 --spawn-rate 5 --run-time 60s --headless
"""

import random
import string

from locust import HttpUser, between, task

WRITTEN_KEYS: list[str] = []
MAX_TRACKED_KEYS = 5000


def random_value(length=8):
    return "".join(random.choices(string.ascii_lowercase, k=length))


def _write(client, name_prefix):
    key = f"key-{random.randint(0, 9999)}"
    resp = client.put(
        f"/v1/keys/{key}",
        json={"value": random_value()},
        name=f"{name_prefix} PUT",
    )
    if resp and resp.status_code == 200:
        if len(WRITTEN_KEYS) < MAX_TRACKED_KEYS:
            WRITTEN_KEYS.append(key)
        else:
            WRITTEN_KEYS[random.randint(0, MAX_TRACKED_KEYS - 1)] = key


def _read(client, name_prefix):
    if not WRITTEN_KEYS:
        return
    key = random.choice(WRITTEN_KEYS)
    client.get(f"/v1/keys/{key}", name=f"{name_prefix} GET")


class SimpleKVSUser(HttpUser):
    """Targets the single-node SimpleKVS baseline."""
    wait_time = between(0.01, 0.05)

    @task(3)
    def write(self):
        _write(self.client, "/v1/keys/[key] SimpleKVS")

    @task(1)
    def read(self):
        _read(self.client, "/v1/keys/[key] SimpleKVS")


class RaftUser(HttpUser):
    """Targets the Raft leader — same request mix as SimpleKVSUser for fair comparison."""
    wait_time = between(0.01, 0.05)

    @task(3)
    def write(self):
        _write(self.client, "/v1/keys/[key] Raft")

    @task(1)
    def read(self):
        _read(self.client, "/v1/keys/[key] Raft")
