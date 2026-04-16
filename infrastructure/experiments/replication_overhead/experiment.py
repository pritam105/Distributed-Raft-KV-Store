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

  # Both in one run using tags (opens browser UI at localhost:8089)
  locust -f experiment.py
"""

import random
import string

from locust import HttpUser, between, task


def random_value(length=8):
    return "".join(random.choices(string.ascii_lowercase, k=length))


class SimpleKVSUser(HttpUser):
    """Targets the single-node SimpleKVS baseline."""
    wait_time = between(0.01, 0.05)

    @task(3)
    def write(self):
        key = f"key-{random.randint(0, 999)}"
        self.client.put(
            f"/v1/keys/{key}",
            json={"value": random_value()},
            name="/v1/keys/[key] PUT",
        )

    @task(1)
    def read(self):
        key = f"key-{random.randint(0, 999)}"
        self.client.get(
            f"/v1/keys/{key}",
            name="/v1/keys/[key] GET",
        )


class RaftUser(HttpUser):
    """Targets the Raft leader — same request mix as SimpleKVSUser for fair comparison."""
    wait_time = between(0.01, 0.05)

    @task(3)
    def write(self):
        key = f"key-{random.randint(0, 999)}"
        self.client.put(
            f"/v1/keys/{key}",
            json={"value": random_value()},
            name="/v1/keys/[key] PUT",
        )

    @task(1)
    def read(self):
        key = f"key-{random.randint(0, 999)}"
        self.client.get(
            f"/v1/keys/{key}",
            name="/v1/keys/[key] GET",
        )
