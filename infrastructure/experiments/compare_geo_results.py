"""
Compare co-located vs geo-distributed experiment CSV results.

Run from infrastructure/experiments:
  python3 compare_geo_results.py \
    --colocated geo_colocated/terraform/colocated_results.csv \
    --distributed geo_distributed/terraform/geo_distributed_results.csv \
    --output geo_tradeoff_report.md

The script prints a summary and optionally writes a small Markdown report.
"""

import argparse
import csv
import statistics
from pathlib import Path


NUMERIC_FIELDS = [
    "write_latency_ms",
    "read_latency_avg_ms",
    "read_latency_min_ms",
    "read_latency_max_ms",
    "convergence_ms",
    "immediate_stale_reads",
    "replica_count",
]


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--colocated",
        default="geo_colocated/terraform/colocated_results.csv",
        help="CSV from geo_colocated experiment",
    )
    parser.add_argument(
        "--distributed",
        default="geo_distributed/terraform/geo_distributed_results.csv",
        help="CSV from geo_distributed experiment",
    )
    parser.add_argument(
        "--output",
        default="",
        help="Optional Markdown report path",
    )
    return parser.parse_args()


def to_float(value):
    if value is None or value == "":
        return None
    try:
        return float(value)
    except ValueError:
        return None


def load_rows(path):
    rows = []
    with open(path, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            parsed = dict(row)
            for field in NUMERIC_FIELDS:
                if field in parsed:
                    parsed[field] = to_float(parsed[field])
            rows.append(parsed)
    return rows


def values(rows, field):
    return [row[field] for row in rows if isinstance(row.get(field), float)]


def percentile(sorted_values, pct):
    if not sorted_values:
        return None
    if len(sorted_values) == 1:
        return sorted_values[0]

    rank = (len(sorted_values) - 1) * pct
    lower = int(rank)
    upper = min(lower + 1, len(sorted_values) - 1)
    weight = rank - lower
    return sorted_values[lower] * (1 - weight) + sorted_values[upper] * weight


def summarize_latency(rows, field):
    vals = sorted(values(rows, field))
    if not vals:
        return {
            "count": 0,
            "avg": None,
            "p50": None,
            "p95": None,
            "min": None,
            "max": None,
        }

    return {
        "count": len(vals),
        "avg": statistics.mean(vals),
        "p50": percentile(vals, 0.50),
        "p95": percentile(vals, 0.95),
        "min": min(vals),
        "max": max(vals),
    }


def summarize_staleness(rows):
    stale = values(rows, "immediate_stale_reads")
    replicas = values(rows, "replica_count")
    stale_total = sum(stale)
    read_total = sum(replicas)
    rate = (stale_total / read_total) if read_total else None
    return {
        "stale_total": stale_total,
        "read_total": read_total,
        "rate": rate,
    }


def successful_rounds(rows):
    return [
        row
        for row in rows
        if row.get("write_status") in ("200", 200, 200.0)
    ]


def fmt_ms(value):
    if value is None:
        return "n/a"
    return f"{value:.2f} ms"


def fmt_rate(value):
    if value is None:
        return "n/a"
    return f"{value * 100:.2f}%"


def ratio(distributed, colocated):
    if distributed is None or colocated in (None, 0):
        return "n/a"
    return f"{distributed / colocated:.2f}x"


def build_summary(name, rows):
    ok_rows = successful_rounds(rows)
    return {
        "name": name,
        "rows": rows,
        "successful_rows": ok_rows,
        "write": summarize_latency(ok_rows, "write_latency_ms"),
        "read": summarize_latency(ok_rows, "read_latency_avg_ms"),
        "convergence": summarize_latency(ok_rows, "convergence_ms"),
        "staleness": summarize_staleness(ok_rows),
    }


def table(summary):
    return [
        [
            summary["name"],
            str(len(summary["successful_rows"])),
            fmt_ms(summary["write"]["avg"]),
            fmt_ms(summary["write"]["p95"]),
            fmt_ms(summary["read"]["avg"]),
            fmt_ms(summary["convergence"]["avg"]),
            fmt_rate(summary["staleness"]["rate"]),
        ]
    ]


def markdown_report(colocated, distributed):
    c_write = colocated["write"]["avg"]
    d_write = distributed["write"]["avg"]
    c_read = colocated["read"]["avg"]
    d_read = distributed["read"]["avg"]
    c_conv = colocated["convergence"]["avg"]
    d_conv = distributed["convergence"]["avg"]

    lines = [
        "# Geo-Distribution Tradeoff Results",
        "",
        "| Deployment | Successful rounds | Avg write | P95 write | Avg read | Avg convergence | Immediate stale rate |",
        "|---|---:|---:|---:|---:|---:|---:|",
    ]
    for row in table(colocated) + table(distributed):
        lines.append("| " + " | ".join(row) + " |")

    lines.extend([
        "",
        "## Comparison",
        "",
        f"- Geo-distributed avg write latency ratio: {ratio(d_write, c_write)}",
        f"- Geo-distributed avg read latency ratio: {ratio(d_read, c_read)}",
        f"- Geo-distributed avg convergence ratio: {ratio(d_conv, c_conv)}",
        "",
        "## Interpretation Guide",
        "",
        "- Higher write latency in the geo-distributed setup indicates the cost of cross-region Raft consensus.",
        "- Lower read latency on nearby replicas would indicate the benefit of client proximity.",
        "- A higher immediate stale rate means followers are more likely to lag right after a write.",
        "- Higher convergence time means replicas take longer to observe the latest committed value.",
        "",
    ])

    return "\n".join(lines)


def print_console_report(colocated, distributed):
    print("=" * 80)
    print("Geo-Distribution Tradeoff Results")
    print("=" * 80)
    print()
    print("Deployment           Rounds  Avg Write   P95 Write   Avg Read    Avg Conv.   Stale")
    print("-" * 80)
    for summary in [colocated, distributed]:
        print(
            f"{summary['name']:<20} "
            f"{len(summary['successful_rows']):>6}  "
            f"{fmt_ms(summary['write']['avg']):>10}  "
            f"{fmt_ms(summary['write']['p95']):>10}  "
            f"{fmt_ms(summary['read']['avg']):>10}  "
            f"{fmt_ms(summary['convergence']['avg']):>10}  "
            f"{fmt_rate(summary['staleness']['rate']):>8}"
        )
    print()
    print("Ratios: geo-distributed / co-located")
    print(f"- Avg write latency: {ratio(distributed['write']['avg'], colocated['write']['avg'])}")
    print(f"- Avg read latency: {ratio(distributed['read']['avg'], colocated['read']['avg'])}")
    print(f"- Avg convergence: {ratio(distributed['convergence']['avg'], colocated['convergence']['avg'])}")


def main():
    args = parse_args()
    colocated_rows = load_rows(args.colocated)
    distributed_rows = load_rows(args.distributed)

    colocated = build_summary("Co-located", colocated_rows)
    distributed = build_summary("Geo-distributed", distributed_rows)

    print_console_report(colocated, distributed)

    if args.output:
        report = markdown_report(colocated, distributed)
        Path(args.output).write_text(report, encoding="utf-8")
        print()
        print(f"Wrote report: {args.output}")


if __name__ == "__main__":
    main()
