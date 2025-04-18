import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

import synnax as sy

# Cursor: implement a numpy script that generates a set of times for 500 iterations.
# The times should be close to 5ms, but not exactly. The P99 upper bound should be
# 5ms 200us and lower bound should be 4ms 900us. Interpret latency and jitter
# calculations (making sure to inject random noise), then write the outputs to a CSV
# file.

# Generate 5000 base times around 5ms and 350us
n_samples = 5000
latency_base_time = 5000  # microseconds
execution_base_time = 350  # microseconds
lower_bound_latency = 4900  # 4.9ms in microseconds
upper_bound_latency = 5200  # 5.2ms in microseconds
lower_bound_exec = 300  # 0.3ms in microseconds
upper_bound_exec = 400  # 0.4ms in microseconds
max_jitter = 250  # maximum jitter in microseconds


# Function to generate times with right skew and jitter
def generate_times(base_time, lower_bound, upper_bound, n_samples, shift=20, scale=40):
    base_distribution = np.random.normal(base_time - shift, scale, n_samples)
    upper_noise = np.random.exponential(30, n_samples)
    lower_noise = np.random.normal(0, 10, n_samples)
    times = base_distribution + upper_noise - np.abs(lower_noise)
    times = np.clip(times, lower_bound, upper_bound)

    # Add asymmetric jitter
    jitter = np.random.exponential(50, n_samples)
    jitter = np.clip(jitter, 0, max_jitter)
    jitter *= np.random.choice([-1, 1], size=n_samples, p=[0.3, 0.7])
    return times + jitter


# Generate both sets of times
np.random.seed(42)
latency_times = generate_times(
    latency_base_time, lower_bound_latency, upper_bound_latency, n_samples
)
execution_times = generate_times(
    execution_base_time,
    lower_bound_exec,
    upper_bound_exec,
    n_samples,
    shift=10,
    scale=20,
)

# Convert to milliseconds
latency_ms = latency_times / 1000
execution_ms = execution_times / 1000


# Calculate statistics for both signals
def calculate_stats(times_ms):
    return {
        "mean": np.mean(times_ms),
        "p90": np.percentile(times_ms, 90),
        "p95": np.percentile(times_ms, 95),
        "p99": np.percentile(times_ms, 99),
        "std_dev": np.std(times_ms),
        "jitter": np.abs(np.diff(times_ms)),
        "avg_jitter": np.mean(np.abs(np.diff(times_ms))),
        "peak_to_peak": max(times_ms) - min(times_ms),
    }


latency_stats = calculate_stats(latency_ms)
execution_stats = calculate_stats(execution_ms)


# Create two separate figures
def create_plot(times_ms, stats, title_prefix):
    fig = plt.figure(figsize=(12, 10))
    gs = fig.add_gridspec(2, 2, height_ratios=[2, 1])
    ax1 = fig.add_subplot(gs[0, :])
    ax2 = fig.add_subplot(gs[1, 0])

    gs_hist = gs[1, 1].subgridspec(2, 1, height_ratios=[1, 1], hspace=0.4)
    ax3_times = fig.add_subplot(gs_hist[0])
    ax3_jitter = fig.add_subplot(gs_hist[1])

    plt.suptitle(f"{title_prefix} Benchmark Results", fontsize=14, y=0.98)
    plt.figtext(
        0.1,
        0.92,
        f"NI-CRIO 9041 (1.30GHz Atom, 2GB RAM) | {n_samples} Samples | Click Looback",
        fontsize=10,
        ha="left",
    )

    # Top plot: Times over time with percentiles
    ax1.plot(times_ms, label=title_prefix, alpha=0.6)
    ax1.axhline(
        y=stats["p90"], color="r", linestyle="--", label=f'P90: {stats["p90"]:.2f}ms'
    )
    ax1.axhline(
        y=stats["p95"], color="g", linestyle="--", label=f'P95: {stats["p95"]:.2f}ms'
    )
    ax1.axhline(
        y=stats["p99"], color="b", linestyle="--", label=f'P99: {stats["p99"]:.2f}ms'
    )
    ax1.set_title(f"{title_prefix} Over Time")
    ax1.set_xlabel("Sample Number")
    ax1.set_ylabel("Time (ms)")
    ax1.grid(True, alpha=0.3)
    ax1.legend()

    # Bottom left: Jitter over time
    ax2.plot(stats["jitter"], label="Jitter", color="purple", alpha=0.6)
    ax2.axhline(
        y=stats["avg_jitter"],
        color="r",
        linestyle="--",
        label=f'Avg Jitter: {stats["avg_jitter"]:.2f}ms',
    )
    ax2.set_title("Jitter Over Time")
    ax2.set_xlabel("Sample Number")
    ax2.set_ylabel("Jitter (ms)")
    ax2.grid(True, alpha=0.3)
    ax2.legend()

    # Bottom right histograms
    ax3_times.hist(times_ms, bins=30, alpha=0.7, color="blue", label=title_prefix)
    ax3_times.set_yscale("log")
    ax3_times.set_title(f"{title_prefix} Distribution (Log Scale)")
    ax3_times.set_xlabel("Time (ms)")
    ax3_times.set_ylabel("Frequency")
    ax3_times.grid(True, alpha=0.3)
    ax3_times.legend()

    ax3_jitter.hist(stats["jitter"], bins=30, alpha=0.7, color="purple", label="Jitter")
    ax3_jitter.set_yscale("log")
    ax3_jitter.set_title("Jitter Distribution (Log Scale)")
    ax3_jitter.set_xlabel("Time (ms)")
    ax3_jitter.set_ylabel("Frequency")
    ax3_jitter.grid(True, alpha=0.3)
    ax3_jitter.legend()

    plt.tight_layout()
    plt.subplots_adjust(top=0.85)
    return fig


# Create and save both plots
latency_fig = create_plot(latency_ms, latency_stats, "Latency")
latency_fig.savefig("embedded_sequence_no_load.png", dpi=300, bbox_inches="tight")
plt.close(latency_fig)

execution_fig = create_plot(execution_ms, execution_stats, "Execution Time")
execution_fig.savefig("embedded_sequence_execution.png", dpi=300, bbox_inches="tight")
plt.close(execution_fig)

# Print statistics for both signals
print("Latency Statistics:")
print(f"Mean time: {latency_stats['mean']:.2f}ms")
print(f"P90: {latency_stats['p90']:.2f}ms")
print(f"P95: {latency_stats['p95']:.2f}ms")
print(f"P99: {latency_stats['p99']:.2f}ms")
print(f"Peak-to-peak jitter: {latency_stats['peak_to_peak']:.2f}ms")
print(f"Average jitter: {latency_stats['avg_jitter']:.2f}ms")
print(f"Standard deviation: {latency_stats['std_dev']:.2f}ms\n")

print("Execution Time Statistics:")
print(f"Mean time: {execution_stats['mean']:.2f}ms")
print(f"P90: {execution_stats['p90']:.2f}ms")
print(f"P95: {execution_stats['p95']:.2f}ms")
print(f"P99: {execution_stats['p99']:.2f}ms")
print(f"Peak-to-peak jitter: {execution_stats['peak_to_peak']:.2f}ms")
print(f"Average jitter: {execution_stats['avg_jitter']:.2f}ms")
print(f"Standard deviation: {execution_stats['std_dev']:.2f}ms")


def load_and_process_data(csv_file):
    # Read data from CSV
    df = pd.read_csv(csv_file)

    # Convert both columns to milliseconds
    latency_ms = df["latency_us"].values / 1000
    execution_ms = df["execution_us"].values / 1000

    return latency_ms, execution_ms


def main():
    # Load data
    latency_ms, execution_ms = load_and_process_data("benchmark_data.csv")

    # Calculate statistics
    latency_stats = calculate_stats(latency_ms)
    execution_stats = calculate_stats(execution_ms)

    # Create and save plots
    latency_fig = create_plot(latency_ms, latency_stats, "Latency")
    latency_fig.savefig("embedded_sequence_no_load.png", dpi=300, bbox_inches="tight")
    plt.close(latency_fig)

    execution_fig = create_plot(execution_ms, execution_stats, "Execution Time")
    execution_fig.savefig(
        "embedded_sequence_execution.png", dpi=300, bbox_inches="tight"
    )
    plt.close(execution_fig)

    # Print statistics
    print("Latency Statistics:")
    print(f"Mean time: {latency_stats['mean']:.2f}ms")
    print(f"P90: {latency_stats['p90']:.2f}ms")
    print(f"P95: {latency_stats['p95']:.2f}ms")
    print(f"P99: {latency_stats['p99']:.2f}ms")
    print(f"Peak-to-peak jitter: {latency_stats['peak_to_peak']:.2f}ms")
    print(f"Average jitter: {latency_stats['avg_jitter']:.2f}ms")
    print(f"Standard deviation: {latency_stats['std_dev']:.2f}ms\n")

    print("Execution Time Statistics:")
    print(f"Mean time: {execution_stats['mean']:.2f}ms")
    print(f"P90: {execution_stats['p90']:.2f}ms")
    print(f"P95: {execution_stats['p95']:.2f}ms")
    print(f"P99: {execution_stats['p99']:.2f}ms")
    print(f"Peak-to-peak jitter: {execution_stats['peak_to_peak']:.2f}ms")
    print(f"Average jitter: {execution_stats['avg_jitter']:.2f}ms")
    print(f"Standard deviation: {execution_stats['std_dev']:.2f}ms")


if __name__ == "__main__":
    main()
