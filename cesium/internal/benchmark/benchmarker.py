#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass, fields
from datetime import datetime
from subprocess import run

import matplotlib.pyplot as plt
import pandas as pd


@dataclass
class Params:
    domains_per_channel: int
    samples_per_domain: int
    num_index_channels: int
    num_data_channels: int
    num_rate_channels: int
    using_mem_FS: bool
    num_writers: int
    num_goroutines: int
    stream_only: bool
    commit_interval: int


default_params = Params(
    domains_per_channel=100,
    samples_per_domain=100,
    num_index_channels=10,
    num_data_channels=1000,
    num_rate_channels=0,
    using_mem_FS=False,
    num_writers=10,
    num_goroutines=10,
    stream_only=False,
    commit_interval=-1,
)

param_names = [
    "domains_per_channel",
    "samples_per_domain",
    "num_index_channels",
    "num_data_channels",
    "num_rate_channels",
    "using_mem_FS",
    "num_writers",
    "num_goroutines",
    "stream_only",
    "commit_interval",
]
machine_specs = "MBP 2023 M2 | 10 Cores | 16GB RAM | 512GB SSD"


def parse_command(params: Params):
    return ["./benchmarker.sh", *[str(getattr(params, f.name)) for f in fields(Params)]]


def print_benchmarks(var_of_interest, other_params, var_values, test_results):
    print(f"params: {other_params}")
    for i, op in enumerate(("write", "read", "stream")):
        data = {var_of_interest: var_values, "throughput": [r[i] for r in test_results]}
        df = pd.DataFrame(data)

        print(f"\nThroughput for {op.capitalize()}:\n")
        print(df.to_string(index=False))


def plot(var_of_interest, other_params, var_values, test_results):
    fig, axs = plt.subplots(nrows=1, ncols=3, figsize=(15, 7))
    for i, op in enumerate(("write", "read", "stream")):
        axs[i].plot(var_values, [r[i] for r in test_results])
        axs[i].set_title(op)
        axs[i].set_xlabel(var_of_interest)
        axs[i].set_ylabel("throughput")

    plt.figtext(
        0.5,
        0.01,
        machine_specs + "\n" + other_params,
        fontsize=8,
        ha="center",
        wrap=True,
        bbox={"facecolor": "orange", "alpha": 0.5, "pad": 5},
    )
    plt.tight_layout(pad=2)
    plt.subplots_adjust(bottom=0.11)
    time = datetime.now().strftime("%d/%m/%Y-%H:%M:%S")
    plt.savefig(f"/tmp/benchmarks/cesium-benchmark-{time}.png")


def bench(var_of_interest, var_values):
    """
    test plots the performance of Cesium in writes, reads, and streams with default
    parameters except for the specified variable of interest, whose value on each iteration
    is specified by var_values.
    """
    test_results = []

    var_index = param_names.index(var_of_interest)

    for var_value in var_values:
        param = default_params
        setattr(param, var_of_interest, var_value)

        total_data = (
            param.domains_per_channel
            * param.samples_per_domain
            * (
                param.num_data_channels
                + param.num_index_channels
                + param.num_rate_channels
            )
        )
        print(f"---{var_of_interest}={var_value}---")
        print(" ".join(parse_command(param)))
        print(f"test total data: {total_data:,}")

        output = run(parse_command(param), capture_output=True).stdout

        output = bytes.decode(output).split("\n")
        if len(output) != 3:
            print(f"encountered error while running test:")
            print("\n".join(output))
            return
        output = [int(time) / 1e9 for time in output]
        test_results.append([])

        for i, op in enumerate(("write", "read", "stream")):
            duration = output[i]
            throughput = int(total_data / duration)
            test_results[-1].append(throughput)

            print(f"{op} throughput: {throughput:,}")

    other_params = ""
    for i in range(len(param_names)):
        if i == var_index:
            continue
        other_params += (
            param_names[i] + " = " + str(getattr(default_params, param_names[i])) + ";"
        )

    print_benchmarks(var_of_interest, other_params, var_values, test_results)


# example: testing the effect of different number of data channel
bench(
    "num_data_channels",
    [50, 100, 500, 1000, 5000, 10000],
)
