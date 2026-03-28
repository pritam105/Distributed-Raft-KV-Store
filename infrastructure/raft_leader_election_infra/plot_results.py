import matplotlib.pyplot as plt
import numpy as np

# Results from our experiment
rounds = [1, 2, 3, 4, 5]
timings = [190, 190, 381, 368, 368]
leaders_crashed = ["nodeA", "nodeB", "nodeA", "nodeC", "nodeB"]
new_leaders = ["nodeB", "nodeA", "nodeC", "nodeB", "nodeC"]

avg = np.mean(timings)
min_t = np.min(timings)
max_t = np.max(timings)

fig, axes = plt.subplots(1, 2, figsize=(14, 6))
fig.suptitle("Raft Leader Election — EC2 Experiment Results (us-west-2)", 
             fontsize=14, fontweight="bold")

# ── Chart 1: Re-election time per round ──────────────────────────────────────
colors = ["#2ecc71" if t < 300 else "#e74c3c" for t in timings]
bars = axes[0].bar(rounds, timings, color=colors, edgecolor="black", linewidth=0.5)
axes[0].axhline(y=avg, color="blue", linestyle="--", linewidth=1.5,
                label=f"Average: {avg:.0f}ms")
axes[0].axhline(y=300, color="orange", linestyle=":", linewidth=1.5,
                label="Election timeout min (300ms)")
axes[0].axhline(y=600, color="red", linestyle=":", linewidth=1.5,
                label="Election timeout max (600ms)")

for bar, t in zip(bars, timings):
    axes[0].text(bar.get_x() + bar.get_width()/2, bar.get_height() + 10,
                 f"{t}ms", ha="center", va="bottom", fontsize=10, fontweight="bold")

axes[0].set_xlabel("Round", fontsize=12)
axes[0].set_ylabel("Re-election Time (ms)", fontsize=12)
axes[0].set_title("Re-election Time per Round", fontsize=12)
axes[0].set_xticks(rounds)
axes[0].set_xticklabels([f"R{r}: {leaders_crashed[i-1]}→{new_leaders[i-1]}" 
                          for r, i in zip(rounds, rounds)],
                         rotation=20, ha="right", fontsize=9)
axes[0].set_ylim(0, 700)
axes[0].legend(fontsize=9)
axes[0].grid(axis="y", alpha=0.3)

# ── Chart 2: Summary stats ────────────────────────────────────────────────────
stats = ["Min", "Average", "Median", "Max"]
values = [190, 299, 368, 381]
colors2 = ["#2ecc71", "#3498db", "#9b59b6", "#e74c3c"]

bars2 = axes[1].bar(stats, values, color=colors2, edgecolor="black", linewidth=0.5)
for bar, v in zip(bars2, values):
    axes[1].text(bar.get_x() + bar.get_width()/2, bar.get_height() + 8,
                 f"{v}ms", ha="center", va="bottom", fontsize=11, fontweight="bold")

axes[1].axhline(y=300, color="orange", linestyle=":", linewidth=1.5,
                label="Timeout min (300ms)")
axes[1].axhline(y=600, color="red", linestyle=":", linewidth=1.5,
                label="Timeout max (600ms)")
axes[1].set_ylabel("Time (ms)", fontsize=12)
axes[1].set_title("Re-election Time Summary", fontsize=12)
axes[1].set_ylim(0, 700)
axes[1].legend(fontsize=9)
axes[1].grid(axis="y", alpha=0.3)

plt.tight_layout()
plt.savefig("infrastructure/raft_leader_election_infra/election_results.png", 
            dpi=150, bbox_inches="tight")
plt.show()
print("Graph saved to election_results.png")