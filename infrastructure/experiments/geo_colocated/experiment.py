"""
Geo-distribution tradeoff measurement script.

Use this against either:
  - geo_colocated Terraform deployment
  - geo_distributed Terraform deployment

It measures:
  - write latency to the current leader
  - read latency from each replica
  - immediate stale read rate
  - convergence time until all replicas return the latest value

Example:
  python3 experiment.py \
    --nodes http://nodeA:8000 http://nodeB:8000 http://nodeC:8000 \
    --rounds 100 \
    --csv results.csv
"""

import argparse
import csv
import statistics
import time
import uuid

import requests


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--nodes", nargs="+", required=True, help="Node base URLs")
    parser.add_argument("--rounds", type=int, default=100)
    parser.add_argument("--timeout", type=float, default=2.0)
    parser.add_argument("--convergence-timeout", type=float, default=5.0)
    parser.add_argument("--poll-interval", type=float, default=0.05)
    parser.add_argument("--csv", default="geo_results.csv")
    return parser.parse_args()


def request_json(method, url, timeout, **kwargs):
    start = time.perf_counter()
    try:
        resp = requests.request(method, url, timeout=timeout, **kwargs)
        elapsed_ms = (time.perf_counter() - start) * 1000
        body = None
        try:
            body = resp.json()
        except ValueError:
            pass
        return resp.status_code, body, elapsed_ms, None
    except requests.RequestException as exc:
        elapsed_ms = (time.perf_counter() - start) * 1000
        return None, None, elapsed_ms, str(exc)


def get_status(node, timeout):
    code, body, latency_ms, err = request_json("GET", f"{node}/status", timeout)
    if code != 200 or not isinstance(body, dict):
        return None, latency_ms, err
    return body, latency_ms, None


def find_leader(nodes, timeout):
    statuses = {}
    for node in nodes:
        status, latency_ms, err = get_status(node, timeout)
        statuses[node] = {
            "status": status,
            "latency_ms": latency_ms,
            "error": err,
        }
        if status and status.get("isLeader"):
            return node, statuses
    return None, statuses


def put_key(leader, key, value, timeout):
    return request_json(
        "PUT",
        f"{leader}/v1/keys/{key}",
        timeout,
        json={"value": value},
    )


def get_key(node, key, timeout):
    code, body, latency_ms, err = request_json(
        "GET",
        f"{node}/v1/keys/{key}",
        timeout,
    )
    value = None
    if isinstance(body, dict):
        value = body.get("value")
    return code, value, latency_ms, err


def wait_until_all_visible(nodes, key, value, timeout, poll_interval, request_timeout):
    start = time.perf_counter()
    deadline = start + timeout
    latest = {}

    while time.perf_counter() < deadline:
        all_visible = True
        for node in nodes:
            code, got, latency_ms, err = get_key(node, key, request_timeout)
            latest[node] = {
                "status": code,
                "value": got,
                "latency_ms": latency_ms,
                "error": err,
            }
            if code != 200 or got != value:
                all_visible = False
        if all_visible:
            return (time.perf_counter() - start) * 1000, latest
        time.sleep(poll_interval)

    return None, latest


def summarize(values):
    values = [v for v in values if v is not None]
    if not values:
        return "n/a"
    return (
        f"avg={statistics.mean(values):.2f}ms "
        f"p50={statistics.median(values):.2f}ms "
        f"min={min(values):.2f}ms "
        f"max={max(values):.2f}ms"
    )


def aggregate(values):
    values = [v for v in values if v is not None]
    if not values:
        return None, None, None
    return statistics.mean(values), min(values), max(values)


def main():
    args = parse_args()
    rows = []

    print("=" * 72)
    print("Geo-distribution tradeoff experiment")
    print("=" * 72)
    print(f"nodes: {', '.join(args.nodes)}")
    print(f"rounds: {args.rounds}")
    print()

    for round_num in range(1, args.rounds + 1):
        leader, statuses = find_leader(args.nodes, args.timeout)
        if not leader:
            print(f"[round {round_num}] no leader found")
            rows.append({
                "round": round_num,
                "leader": "",
                "write_status": "",
                "write_latency_ms": "",
                "immediate_stale_reads": "",
                "replica_count": len(args.nodes),
                "convergence_ms": "",
                "error": "no leader",
            })
            time.sleep(0.25)
            continue

        key = f"geo-{round_num}-{uuid.uuid4().hex[:8]}"
        value = f"value-{uuid.uuid4().hex}"

        write_code, _, write_latency_ms, write_err = put_key(
            leader, key, value, args.timeout
        )

        immediate_stale = 0
        read_latencies = []
        for node in args.nodes:
            code, got, read_latency_ms, _ = get_key(node, key, args.timeout)
            read_latencies.append(read_latency_ms)
            if code != 200 or got != value:
                immediate_stale += 1
        read_avg, read_min, read_max = aggregate(read_latencies)

        convergence_ms, _ = wait_until_all_visible(
            args.nodes,
            key,
            value,
            args.convergence_timeout,
            args.poll_interval,
            args.timeout,
        )

        rows.append({
            "round": round_num,
            "leader": leader,
            "write_status": write_code,
            "write_latency_ms": write_latency_ms,
            "immediate_stale_reads": immediate_stale,
            "replica_count": len(args.nodes),
            "read_latency_avg_ms": read_avg,
            "read_latency_min_ms": read_min,
            "read_latency_max_ms": read_max,
            "convergence_ms": convergence_ms,
            "error": write_err or "",
        })

        print(
            f"[round {round_num}] leader={leader} "
            f"write={write_code} {write_latency_ms:.2f}ms "
            f"read_avg={read_avg if read_avg is not None else 'n/a'}ms "
            f"stale={immediate_stale}/{len(args.nodes)} "
            f"convergence={convergence_ms if convergence_ms is not None else 'timeout'}ms"
        )

    with open(args.csv, "w", newline="", encoding="utf-8") as f:
        fieldnames = [
            "round",
            "leader",
            "write_status",
            "write_latency_ms",
            "immediate_stale_reads",
            "replica_count",
            "read_latency_avg_ms",
            "read_latency_min_ms",
            "read_latency_max_ms",
            "convergence_ms",
            "error",
        ]
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows)

    write_latencies = [
        row["write_latency_ms"]
        for row in rows
        if isinstance(row["write_latency_ms"], (int, float))
    ]
    convergence = [
        row["convergence_ms"]
        for row in rows
        if isinstance(row["convergence_ms"], (int, float))
    ]
    read_latencies = [
        row["read_latency_avg_ms"]
        for row in rows
        if isinstance(row.get("read_latency_avg_ms"), (int, float))
    ]
    stale_total = sum(
        row["immediate_stale_reads"]
        for row in rows
        if isinstance(row["immediate_stale_reads"], int)
    )
    read_attempts = sum(
        row["replica_count"]
        for row in rows
        if isinstance(row["replica_count"], int)
    )

    print()
    print("=" * 72)
    print("Summary")
    print("=" * 72)
    print(f"write latency: {summarize(write_latencies)}")
    print(f"read latency: {summarize(read_latencies)}")
    print(f"convergence: {summarize(convergence)}")
    if read_attempts:
        print(f"immediate stale reads: {stale_total}/{read_attempts}")
    print(f"csv: {args.csv}")


if __name__ == "__main__":
    main()
