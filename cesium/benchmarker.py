from subprocess import run

import matplotlib.pyplot as plt

param_names = ["domains_per_channel", "samples_per_domain", "num_index_channels",
               "num_data_channels", "num_rate_channels", "using_mem_FS", "num_writers",
               "num_goroutines", "stream_only", "commit_interval"]
machine_specs = "MBP 2023 M2 | 10 Cores | 16GB RAM | 512GB SSD"


def parse_command(params):
    return ["./benchmarker.sh", *[str(arg) for arg in params]]


def plot(var_of_interest, other_params, var_values, test_results):
    fig, axs = plt.subplots(nrows=1, ncols=3, figsize=(15, 7))
    for i, op in enumerate(("write", "read", "stream")):
        axs[i].plot(var_values, [r[i] for r in test_results])
        axs[i].set_title(op)
        axs[i].set_xlabel(var_of_interest)
        axs[i].set_ylabel("throughput")

    plt.figtext(0.5, 0.01, machine_specs + "\n" + other_params,
                fontsize=8, ha='center', wrap=True,
                bbox={"facecolor": "orange", "alpha": 0.5, "pad": 5})
    plt.tight_layout(pad=2)
    plt.subplots_adjust(bottom=0.11)
    plt.show()


def bench(var_of_interest, var_values, default_params):
    """
    test plots the performance of Cesium in writes, reads, and streams with default
    parameters except for the specified variable of interest, whose value on each iteration
    is specified by var_values.
    """
    test_results = []

    var_index = param_names.index(var_of_interest)

    for var_value in var_values:
        param = default_params
        param[var_index] = var_value

        total_data = param[0] * param[1] * (param[2] + param[3] + param[4])
        print(f"---{var_of_interest}={var_value}---")
        print(" ".join(parse_command(param)))
        print(f"test total data: {total_data:,}")

        output = run(parse_command(param),
                     capture_output=True).stdout

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
        other_params += param_names[i] + " = " + str(default_params[i]) + ";"

    plot(var_of_interest, other_params, var_values, test_results)


# example: testing the effect of different number of commit_intervals
bench("num_data_channels",
      [1000, 3000, 5000, 7000, 9000, 11000, 13000, 15000, 17000, 19000],
      [100, 100, 10, 1000, 0, "false", 1, 8, "false", -1]
      )
