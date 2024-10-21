from bench.bench_synnax import bench_synnax
from bench.config import TestConfig

if __name__ == "__main__":
    print("Synnax:")
    cfg = TestConfig()
    print(bench_synnax(cfg))

    # print("Influx:")
    # print(bench_influx())

    # print("Timescale:")
    # print(bench_timescale())

