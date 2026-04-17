"""
Continuous write load during leader failover experiment.
Run this while experiment.py crashes/restarts nodes.
Watch for 503 errors during re-election gap, then recovery.

Run:
  locust -f locust_election.py RaftUser \
    --host http://<leader-ip>:8000 \
    --users 20 --spawn-rate 2 --run-time 120s
"""

import random
import string
from locust import HttpUser, between, task


def random_value(n=8):
    return "".join(random.choices(string.ascii_lowercase, k=n))


class RaftUser(HttpUser):
    wait_time = between(0.05, 0.1)

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
