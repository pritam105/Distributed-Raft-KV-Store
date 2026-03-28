import requests
import time
import subprocess
import json
import statistics
import sys
import argparse

def parse_args():
    parser = argparse.ArgumentParser(description="Raft election timing experiment")
    parser.add_argument("--node-a", required=True, help="nodeA public IP")
    parser.add_argument("--node-b", required=True, help="nodeB public IP")
    parser.add_argument("--node-c", required=True, help="nodeC public IP")
    parser.add_argument("--key", required=True, help="path to .pem key file")
    parser.add_argument("--port", default=7000, type=int)
    parser.add_argument("--rounds", default=5, type=int)
    return parser.parse_args()

args = parse_args()

NODES = {
    "nodeA": args.node_a,
    "nodeB": args.node_b,
    "nodeC": args.node_c,
}
PORT = args.port
KEY = args.key
ROUNDS = args.rounds

def get_status(ip):
    try:
        r = requests.get(f"http://{ip}:{PORT}/status", timeout=2)
        return r.json()
    except Exception:
        return None


def get_metrics(ip):
    try:
        r = requests.get(f"http://{ip}:{PORT}/metrics", timeout=2)
        return r.json()
    except Exception:
        return None


def find_leader():
    for node_id, ip in NODES.items():
        s = get_status(ip)
        if s and s.get("isLeader"):
            return node_id, ip
    return None, None


def crash_node(ip):
    subprocess.run(
        ["ssh", "-i", KEY, "-o", "StrictHostKeyChecking=no",
         f"ec2-user@{ip}", "sudo systemctl stop raft-node"],
        capture_output=True
    )


def restart_node(ip):
    subprocess.run(
        ["ssh", "-i", KEY, "-o", "StrictHostKeyChecking=no",
         f"ec2-user@{ip}", "sudo systemctl start raft-node"],
        capture_output=True
    )


def wait_for_new_leader(crashed_ip, timeout=10):
    start = time.time()
    while time.time() - start < timeout:
        for node_id, ip in NODES.items():
            if ip == crashed_ip:
                continue
            s = get_status(ip)
            if s and s.get("isLeader"):
                elapsed = (time.time() - start) * 1000
                return node_id, s.get("term"), elapsed
        time.sleep(0.05)
    return None, None, None


def print_separator():
    print("-" * 50)


def run_experiment():
    print("=" * 50)
    print("  RAFT LEADER ELECTION EXPERIMENT")
    print("=" * 50)

    # Check all nodes are up
    print("\nChecking all nodes...")
    for node_id, ip in NODES.items():
        s = get_status(ip)
        if s:
            print(f"  {node_id} ({ip}) → {s['state']} term={s['term']}")
        else:
            print(f"  {node_id} ({ip}) → UNREACHABLE")
            sys.exit(1)

    timings = []

    for round_num in range(1, ROUNDS + 1):
        print_separator()
        print(f"Round {round_num}/{ROUNDS}")

        # Find current leader
        leader_id, leader_ip = find_leader()
        if not leader_id:
            print("  No leader found — skipping")
            continue

        s = get_status(leader_ip)
        print(f"  Current leader : {leader_id} ({leader_ip})")
        print(f"  Current term   : {s['term']}")

        # Crash the leader
        print(f"  Crashing {leader_id}...")
        start = time.time()
        crash_node(leader_ip)

        # Wait for new leader
        new_leader_id, new_term, elapsed_ms = wait_for_new_leader(leader_ip)

        if new_leader_id:
            timings.append(elapsed_ms)
            print(f"  New leader     : {new_leader_id}")
            print(f"  New term       : {new_term}")
            print(f"  Re-election    : {elapsed_ms:.0f}ms")
        else:
            print("  No new leader found within timeout")

        # Restart crashed node
        print(f"  Restarting {leader_id}...")
        restart_node(leader_ip)
        time.sleep(4)

    # Summary
    print_separator()
    print("\nRESULTS SUMMARY")
    print_separator()
    print(f"Rounds completed  : {len(timings)}/{ROUNDS}")
    if timings:
        print(f"Min re-election   : {min(timings):.0f}ms")
        print(f"Max re-election   : {max(timings):.0f}ms")
        print(f"Average           : {statistics.mean(timings):.0f}ms")
        print(f"Median            : {statistics.median(timings):.0f}ms")
        if len(timings) > 1:
            print(f"Std deviation     : {statistics.stdev(timings):.0f}ms")
        print(f"\nAll timings (ms)  : {[f'{t:.0f}' for t in timings]}")

    # Final metrics
    print("\nFINAL METRICS")
    print_separator()
    for node_id, ip in NODES.items():
        m = get_metrics(ip)
        if m:
            print(f"\n  {node_id}:")
            print(f"    state            : {m['state']}")
            print(f"    term             : {m['term']}")
            print(f"    electionsStarted : {m['electionsStarted']}")
            print(f"    heartbeatsSent   : {m['heartbeatsSent']}")
            print(f"    heartbeatsRecvd  : {m['heartbeatsRecvd']}")
            print(f"    votesGranted     : {m['votesGranted']}")
            print(f"    currentLeaderID  : {m['currentLeaderID']}")


if __name__ == "__main__":
    run_experiment()