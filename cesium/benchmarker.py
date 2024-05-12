from subprocess import run

import matplotlib.pyplot as plt

param_names = ["domains_per_channel", "samples_per_domain", "num_index_channels",
               "num_data_channels", "num_rate_channels", "using_mem_FS", "num_writers",
               "num_goroutines", "stream_only", "commit_interval"]
default_params = [100, 1000, 10, 1000, 0, "false", 1, 8, "false", -1]


def parse_command(params):
    return ["./benchmarker.sh", *[str(arg) for arg in params]]


def plot(var_of_interest, var_values, test_results):
    fig, axs = plt.subplots(nrows=1, ncols=3)
    for i, op in enumerate(("write", "read", "stream")):
        axs[i].plot(var_values, [r[i] for r in test_results])
        axs[i].set_title(op)
        axs[i].xlabel(var_of_interest)
    fig.tight_layout()
    plt.show()


def test(var_of_interest, var_values):
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

        print(output)
        output = bytes.decode(output).split("\n")
        output = [int(time) / 1e9 for time in output]
        test_results.append([])

        for i, op in enumerate(("write", "read", "stream")):
            duration = output[i]
            throughput = int(total_data / duration)
            test_results[-1].append(throughput)

            print(f"{op} throughput: {throughput:,}")

    plot(var_of_interest, var_values, test_results)


# example: testing the effect of different number of writers
test("num_writers", [1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
