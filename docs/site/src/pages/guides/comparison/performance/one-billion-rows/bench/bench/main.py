#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

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

