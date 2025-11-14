#!/usr/bin/env python3
"""
Plot runtime results and jitter analysis from runtime_results.csv
"""

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from matplotlib.gridspec import GridSpec

# Set style for better-looking plots
plt.style.use('seaborn-v0_8-darkgrid')
plt.rcParams['font.size'] = 10
plt.rcParams['axes.labelsize'] = 11
plt.rcParams['axes.titlesize'] = 12
plt.rcParams['legend.fontsize'] = 9

# Read the CSV file
df = pd.read_csv("runtime_results.csv")

# Convert nanoseconds to microseconds for better readability
df["elapsed_us"] = df["elapsed_ns"] / 1_000
df["elapsed_ms"] = df["elapsed_ns"] / 1_000_000

# Calculate inter-iteration time (time between iterations)
df["inter_iteration_ns"] = df["elapsed_ns"].diff()
df["inter_iteration_us"] = df["inter_iteration_ns"] / 1_000
df["inter_iteration_ms"] = df["inter_iteration_ns"] / 1_000_000

# Calculate statistics
jitter_ns = df["inter_iteration_ns"].std()
jitter_us = jitter_ns / 1_000
jitter_ms = jitter_ns / 1_000_000
mean_inter_iteration_ns = df["inter_iteration_ns"].mean()
mean_inter_iteration_us = mean_inter_iteration_ns / 1_000
mean_inter_iteration_ms = mean_inter_iteration_us / 1_000
cv = (jitter_ns / mean_inter_iteration_ns) * 100 if mean_inter_iteration_ns != 0 else 0  # Coefficient of variation (%)

# Calculate percentiles
p50 = df["inter_iteration_us"].quantile(0.50)
p90 = df["inter_iteration_us"].quantile(0.90)
p95 = df["inter_iteration_us"].quantile(0.95)
p99 = df["inter_iteration_us"].quantile(0.99)

# Calculate rolling statistics (window of 3 for smoothing)
if len(df) > 3:
    df["rolling_mean"] = df["inter_iteration_us"].rolling(window=3, center=True).mean()
    df["rolling_std"] = df["inter_iteration_us"].rolling(window=3, center=True).std()

# Calculate deviation from mean
df["deviation_us"] = df["inter_iteration_us"] - mean_inter_iteration_us
df["deviation_pct"] = (df["deviation_us"] / mean_inter_iteration_us) * 100

print("=" * 70)
print("RUNTIME PERFORMANCE STATISTICS")
print("=" * 70)
print(f"\nTiming Statistics (microseconds):")
print(f"  Mean inter-iteration time: {mean_inter_iteration_us:.2f} µs")
print(f"  Jitter (std dev):          {jitter_us:.2f} µs")
print(f"  Coefficient of variation:  {cv:.2f}%")
print(f"  Min inter-iteration time:  {df['inter_iteration_us'].min():.2f} µs")
print(f"  Max inter-iteration time:  {df['inter_iteration_us'].max():.2f} µs")
print(f"\nPercentiles (microseconds):")
print(f"  50th (median):             {p50:.2f} µs")
print(f"  90th:                      {p90:.2f} µs")
print(f"  95th:                      {p95:.2f} µs")
print(f"  99th:                      {p99:.2f} µs")
print(f"\nTotal iterations:            {len(df)}")
print(f"Total runtime:               {df['elapsed_ms'].iloc[-1]:.3f} ms")
print("=" * 70)

# Create figure with custom grid layout
fig = plt.figure(figsize=(16, 12))
gs = GridSpec(3, 3, figure=fig, hspace=0.35, wspace=0.3)

fig.suptitle("Arc Runtime Performance Analysis", fontsize=18, fontweight="bold", y=0.995)

# Plot 1: Inter-iteration time with rolling average
ax1 = fig.add_subplot(gs[0, :2])
ax1.plot(
    df["iteration"][1:],
    df["inter_iteration_us"][1:],
    marker="o",
    markersize=4,
    linestyle="-",
    linewidth=1.5,
    color="#2E86AB",
    label="Inter-iteration time",
    alpha=0.7
)
if len(df) > 3:
    ax1.plot(
        df["iteration"][1:],
        df["rolling_mean"][1:],
        linestyle="-",
        linewidth=2.5,
        color="#A23B72",
        label="Rolling mean (n=3)",
        alpha=0.9
    )
ax1.axhline(
    y=mean_inter_iteration_us,
    color="#F18F01",
    linestyle="--",
    label=f"Mean: {mean_inter_iteration_us:.2f} µs",
    linewidth=2
)
ax1.fill_between(
    df["iteration"][1:],
    mean_inter_iteration_us - jitter_us,
    mean_inter_iteration_us + jitter_us,
    alpha=0.15,
    color="#F18F01",
    label=f"±1σ: {jitter_us:.2f} µs",
)
ax1.set_xlabel("Iteration", fontweight="bold")
ax1.set_ylabel("Time (µs)", fontweight="bold")
ax1.set_title("Inter-Iteration Timing with Jitter Band", fontweight="bold", pad=10)
ax1.legend(loc="best", framealpha=0.9)
ax1.grid(True, alpha=0.3, linestyle=":")

# Plot 2: Statistics summary box
ax2 = fig.add_subplot(gs[0, 2])
ax2.axis('off')
stats_text = f"""
TIMING STATISTICS

Mean:        {mean_inter_iteration_us:.2f} µs
Std Dev:     {jitter_us:.2f} µs
CV:          {cv:.2f}%

Min:         {df['inter_iteration_us'].min():.2f} µs
Max:         {df['inter_iteration_us'].max():.2f} µs
Range:       {df['inter_iteration_us'].max() - df['inter_iteration_us'].min():.2f} µs

PERCENTILES

p50:         {p50:.2f} µs
p90:         {p90:.2f} µs
p95:         {p95:.2f} µs
p99:         {p99:.2f} µs

Iterations:  {len(df)}
Total Time:  {df['elapsed_ms'].iloc[-1]:.3f} ms
"""
ax2.text(
    0.1, 0.95, stats_text,
    transform=ax2.transAxes,
    fontsize=10,
    verticalalignment="top",
    horizontalalignment="left",
    fontfamily="monospace",
    bbox=dict(boxstyle="round,pad=1", facecolor="#E8F4F8", edgecolor="#2E86AB", linewidth=2, alpha=0.9)
)

# Plot 3: Histogram with percentile markers
ax3 = fig.add_subplot(gs[1, 0])
n, bins, patches = ax3.hist(
    df["inter_iteration_us"][1:].dropna(),
    bins=15,
    edgecolor="black",
    alpha=0.7,
    color="#2E86AB"
)
ax3.axvline(x=mean_inter_iteration_us, color="#F18F01", linestyle="--", label="Mean", linewidth=2.5)
ax3.axvline(x=p50, color="#A23B72", linestyle=":", label="Median (p50)", linewidth=2)
ax3.axvline(x=p95, color="#C73E1D", linestyle=":", label="p95", linewidth=2)
ax3.set_xlabel("Inter-Iteration Time (µs)", fontweight="bold")
ax3.set_ylabel("Frequency", fontweight="bold")
ax3.set_title("Distribution of Timing", fontweight="bold", pad=10)
ax3.legend(loc="best", framealpha=0.9)
ax3.grid(True, alpha=0.3, axis="y", linestyle=":")

# Plot 4: Cumulative Distribution Function (CDF)
ax4 = fig.add_subplot(gs[1, 1])
sorted_data = np.sort(df["inter_iteration_us"][1:].dropna())
cdf = np.arange(1, len(sorted_data) + 1) / len(sorted_data)
ax4.plot(sorted_data, cdf * 100, linewidth=2.5, color="#2E86AB")
ax4.axhline(y=50, color="#A23B72", linestyle=":", alpha=0.6, linewidth=1.5)
ax4.axhline(y=90, color="#F18F01", linestyle=":", alpha=0.6, linewidth=1.5)
ax4.axhline(y=95, color="#C73E1D", linestyle=":", alpha=0.6, linewidth=1.5)
ax4.axvline(x=p50, color="#A23B72", linestyle=":", alpha=0.6, linewidth=1.5, label=f"p50: {p50:.2f} µs")
ax4.axvline(x=p90, color="#F18F01", linestyle=":", alpha=0.6, linewidth=1.5, label=f"p90: {p90:.2f} µs")
ax4.axvline(x=p95, color="#C73E1D", linestyle=":", alpha=0.6, linewidth=1.5, label=f"p95: {p95:.2f} µs")
ax4.set_xlabel("Inter-Iteration Time (µs)", fontweight="bold")
ax4.set_ylabel("Cumulative Probability (%)", fontweight="bold")
ax4.set_title("Cumulative Distribution Function", fontweight="bold", pad=10)
ax4.legend(loc="lower right", framealpha=0.9)
ax4.grid(True, alpha=0.3, linestyle=":")

# Plot 5: Box plot with outliers
ax5 = fig.add_subplot(gs[1, 2])
box_data = df["inter_iteration_us"][1:].dropna()
bp = ax5.boxplot(
    [box_data],
    vert=True,
    patch_artist=True,
    labels=["Inter-Iteration"],
    showfliers=True,
    widths=0.6
)
bp["boxes"][0].set_facecolor("#2E86AB")
bp["boxes"][0].set_alpha(0.6)
bp["medians"][0].set_color("#C73E1D")
bp["medians"][0].set_linewidth(2.5)
for flier in bp["fliers"]:
    flier.set_marker("o")
    flier.set_markerfacecolor("#F18F01")
    flier.set_markersize(6)
    flier.set_alpha(0.6)
ax5.set_ylabel("Time (µs)", fontweight="bold")
ax5.set_title("Box Plot with Outliers", fontweight="bold", pad=10)
ax5.grid(True, alpha=0.3, axis="y", linestyle=":")

# Plot 6: Deviation from mean
ax6 = fig.add_subplot(gs[2, :2])
colors = ["#2E86AB" if abs(x) <= jitter_us else "#F18F01" for x in df["deviation_us"][1:]]
ax6.bar(
    df["iteration"][1:],
    df["deviation_us"][1:],
    color=colors,
    alpha=0.7,
    edgecolor="black",
    linewidth=0.5
)
ax6.axhline(y=0, color="black", linestyle="-", linewidth=1.5)
ax6.axhline(y=jitter_us, color="#C73E1D", linestyle="--", alpha=0.6, linewidth=1.5, label=f"+1σ: {jitter_us:.2f} µs")
ax6.axhline(y=-jitter_us, color="#C73E1D", linestyle="--", alpha=0.6, linewidth=1.5, label=f"-1σ: {-jitter_us:.2f} µs")
ax6.set_xlabel("Iteration", fontweight="bold")
ax6.set_ylabel("Deviation from Mean (µs)", fontweight="bold")
ax6.set_title("Timing Deviation from Mean (Blue: within ±1σ, Orange: outliers)", fontweight="bold", pad=10)
ax6.legend(loc="best", framealpha=0.9)
ax6.grid(True, alpha=0.3, axis="y", linestyle=":")

# Plot 7: Cumulative elapsed time
ax7 = fig.add_subplot(gs[2, 2])
ax7.plot(df["iteration"], df["elapsed_ms"], marker="o", markersize=4, linestyle="-", linewidth=2, color="#2E86AB")
ax7.fill_between(df["iteration"], 0, df["elapsed_ms"], alpha=0.2, color="#2E86AB")
ax7.set_xlabel("Iteration", fontweight="bold")
ax7.set_ylabel("Cumulative Time (ms)", fontweight="bold")
ax7.set_title("Total Elapsed Time", fontweight="bold", pad=10)
ax7.grid(True, alpha=0.3, linestyle=":")

plt.savefig("runtime_performance.png", dpi=300, bbox_inches="tight")
print(f"\nPlot saved to runtime_performance.png")
plt.show()