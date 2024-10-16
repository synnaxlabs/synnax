from bench.bench_influx import bench_influx
from bench.bench_synnax import bench_synnax
from bench.bench_timescale import bench_timescale

if __name__ == "__main__":
    print("Synnax:")
    print(bench_synnax())

    print("Influx:")
    print(bench_influx())

    print("Timescale:")
    print(bench_timescale())

