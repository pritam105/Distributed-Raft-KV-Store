"""
Leader Election + Failover Experiment
Uses AWS CLI to stop/start instances instead of SSH.

Run:
  python3 experiment.py \
    --node-a 34.207.123.151 \
    --node-b 54.205.141.246 \
    --node-c 184.73.6.31 \
    --instance-a i-0c1354fe989dd519a \
    --instance-b i-07108eec1f39ebea9 \
    --instance-c i-087722d52ef01a60f \
    --region us-east-1 \
    --port 8000 \
    --rounds 3
"""

import time
import subprocess
import requests
import random
import string
import argparse
import statistics


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--node-a",      required=True)
    parser.add_argument("--node-b",      required=True)
    parser.add_argument("--node-c",      required=True)
    parser.add_argument("--instance-a",  required=True)
    parser.add_argument("--instance-b",  required=True)
    parser.add_argument("--instance-c",  required=True)
    parser.add_argument("--region",      default="us-east-1")
    parser.add_argument("--port",        default=8000, type=int)
    parser.add_argument("--rounds",      default=3, type=int)
    return parser.parse_args()


def get_status(ip, port):
    try:
        r = requests.get(f"http://{ip}:{port}/status", timeout=2)
        return r.json()
    except Exception:
        return None


def find_leader(nodes, port):
    for node_id, ip in nodes.items():
        s = get_status(ip, port)
        if s and s.get("isLeader"):
            return node_id, ip
    return None, None


def stop_instance(instance_id, region):
    subprocess.run(
        ["aws", "ec2", "stop-instances",
         "--region", region,
         "--instance-ids", instance_id],
        capture_output=True
    )


def start_instance(instance_id, region):
    subprocess.run(
        ["aws", "ec2", "start-instances",
         "--region", region,
         "--instance-ids", instance_id],
        capture_output=True
    )


def get_public_ip(instance_id, region):
    result = subprocess.run(
        ["aws", "ec2", "describe-instances",
         "--region", region,
         "--instance-ids", instance_id,
         "--query", "Reservations[0].Instances[0].PublicIpAddress",
         "--output", "text"],
        capture_output=True, text=True
    )
    ip = result.stdout.strip()
    return ip if ip != "None" else None


def wait_for_new_leader(nodes, crashed_ip, port, timeout=15):
    start = time.time()
    while time.time() - start < timeout:
        for node_id, ip in nodes.items():
            if ip == crashed_ip:
                continue
            s = get_status(ip, port)
            if s and s.get("isLeader"):
                elapsed = (time.time() - start) * 1000
                return node_id, s.get("term"), elapsed
        time.sleep(0.05)
    return None, None, None


def random_value(n=8):
    return "".join(random.choices(string.ascii_lowercase, k=n))


def write_key(ip, port, key, value):
    try:
        r = requests.put(
            f"http://{ip}:{port}/v1/keys/{key}",
            json={"value": value},
            timeout=3
        )
        return r.status_code == 200
    except Exception:
        return False


def read_key(ip, port, key):
    try:
        r = requests.get(f"http://{ip}:{port}/v1/keys/{key}", timeout=3)
        if r.status_code == 200:
            return r.json().get("value")
        return None
    except Exception:
        return None


def sep():
    print("-" * 60)


def run_experiment():
    args = parse_args()

    NODES = {
        "nodeA": args.node_a,
        "nodeB": args.node_b,
        "nodeC": args.node_c,
    }
    INSTANCES = {
        "nodeA": args.instance_a,
        "nodeB": args.instance_b,
        "nodeC": args.instance_c,
    }
    PORT    = args.port
    REGION  = args.region
    ROUNDS  = args.rounds

    print("=" * 60)
    print("  RAFT LEADER ELECTION + FAILOVER EXPERIMENT")
    print("=" * 60)

    # Check all nodes up
    print("\n[1] Checking all nodes...")
    for nid, ip in NODES.items():
        s = get_status(ip, PORT)
        if s:
            print(f"    {nid} ({ip}) → {s['state']} term={s['term']}")
        else:
            print(f"    {nid} ({ip}) → UNREACHABLE")
            return

    election_times = []
    data_loss_counts = []
    catchup_counts = []

    for round_num in range(1, ROUNDS + 1):
        sep()
        print(f"ROUND {round_num}/{ROUNDS}")
        sep()

        # Find current leader
        leader_id, leader_ip = find_leader(NODES, PORT)
        if not leader_id:
            print("  No leader found — skipping")
            continue

        leader_instance = INSTANCES[leader_id]
        s = get_status(leader_ip, PORT)
        print(f"  Leader       : {leader_id} ({leader_ip})")
        print(f"  Instance     : {leader_instance}")
        print(f"  Term         : {s['term']}")

        # Phase 1: Write 10 keys before crash
        print(f"\n  Phase 1 — writing 10 keys before crash...")
        pre_keys = {}
        for i in range(10):
            k = f"r{round_num}-pre-{i}"
            v = random_value()
            if write_key(leader_ip, PORT, k, v):
                pre_keys[k] = v
        print(f"    Written: {len(pre_keys)}/10")

        # Phase 2: Stop instance and measure re-election
        print(f"\n  Phase 2 — stopping {leader_id} via AWS CLI...")
        start = time.time()
        stop_instance(leader_instance, REGION)

        new_leader_id, new_term, elapsed_ms = wait_for_new_leader(
            NODES, leader_ip, PORT
        )

        if not new_leader_id:
            print("  ERROR: no new leader found within timeout")
            # restart the node before next round
            start_instance(leader_instance, REGION)
            time.sleep(30)
            continue

        election_times.append(elapsed_ms)
        print(f"    New leader   : {new_leader_id}")
        print(f"    New term     : {new_term}")
        print(f"    Re-election  : {elapsed_ms:.0f}ms")

        # Phase 3: Verify pre-crash data survived
        print(f"\n  Phase 3 — verifying pre-crash data survived...")
        new_leader_ip = NODES[new_leader_id]
        survived = sum(1 for k, v in pre_keys.items()
                       if read_key(new_leader_ip, PORT, k) == v)
        lost = len(pre_keys) - survived
        data_loss_counts.append(lost)
        print(f"    Survived : {survived}/{len(pre_keys)}")
        print(f"    Lost     : {lost}/{len(pre_keys)}")
        if lost == 0:
            print(f"    ✓ NO DATA LOSS")
        else:
            print(f"    ✗ DATA LOSS DETECTED")

        # Phase 4: Write 10 more keys to new leader
        print(f"\n  Phase 4 — writing 10 keys to new leader...")
        post_keys = {}
        for i in range(10):
            k = f"r{round_num}-post-{i}"
            v = random_value()
            if write_key(new_leader_ip, PORT, k, v):
                post_keys[k] = v
        print(f"    Written: {len(post_keys)}/10")

        # Phase 5: Restart crashed instance
        print(f"\n  Phase 5 — restarting {leader_id}...")
        start_instance(leader_instance, REGION)

        # Wait for instance to get public IP and boot
        print(f"    Waiting for {leader_id} to boot...")
        new_ip = None
        for _ in range(24):  # wait up to 2 minutes
            time.sleep(5)
            new_ip = get_public_ip(leader_instance, REGION)
            if new_ip:
                print(f"    {leader_id} new IP: {new_ip}")
                NODES[leader_id] = new_ip
                break

        if not new_ip:
            print(f"    {leader_id} did not get public IP within timeout")
            continue

        # Wait for node to be ready
        print(f"    Waiting for Raft to start on {leader_id}...")
        for _ in range(12):
            time.sleep(5)
            s = get_status(new_ip, PORT)
            if s:
                print(f"    {leader_id} → {s['state']} term={s['term']}")
                if s['state'] == 'Follower':
                    print(f"    ✓ Rejoined as Follower")
                break

        # Phase 6: Verify rejoined node caught up
        print(f"\n  Phase 6 — verifying {leader_id} caught up...")
        time.sleep(5)
        caught = sum(1 for k, v in post_keys.items()
                     if read_key(new_ip, PORT, k) == v)
        catchup_counts.append(caught)
        print(f"    Post-crash keys visible: {caught}/{len(post_keys)}")
        if caught == len(post_keys):
            print(f"    ✓ Full catchup via log replication")
        else:
            print(f"    ✗ Partial catchup: {len(post_keys)-caught} missing")

        time.sleep(3)

    # Summary
    sep()
    print("\nRESULTS SUMMARY")
    sep()
    print(f"Rounds completed    : {len(election_times)}/{ROUNDS}")
    if election_times:
        print(f"Re-election min     : {min(election_times):.0f}ms")
        print(f"Re-election max     : {max(election_times):.0f}ms")
        print(f"Re-election avg     : {statistics.mean(election_times):.0f}ms")
        if len(election_times) > 1:
            print(f"Std deviation       : {statistics.stdev(election_times):.0f}ms")
        print(f"Total data loss     : {sum(data_loss_counts)} keys")
        if catchup_counts:
            print(f"Avg catchup         : {statistics.mean(catchup_counts):.1f}/10 keys")

    print("\nFinal cluster state:")
    for nid, ip in NODES.items():
        s = get_status(ip, PORT)
        if s:
            print(f"  {nid}: {s['state']} term={s['term']} leader={s['leaderID']}")
        else:
            print(f"  {nid}: UNREACHABLE")


if __name__ == "__main__":
    run_experiment()
