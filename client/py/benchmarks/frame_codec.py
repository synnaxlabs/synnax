#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Benchmarks of the frame codec.

Usage: uv run python benchmarks/frame_codec.py

Prints one line per benchmark in the Go benchmark format, which
scripts/compare_benchmarks.py reads.
"""

import timeit
from collections.abc import Callable

import numpy as np

import synnax as sy
from synnax.framer.codec import Codec


def bench(name: str, fn: Callable[[], object]) -> None:
    """Prints the fastest of five timings of fn as a Go benchmark line."""
    timer = timeit.Timer(fn)
    loops, _ = timer.autorange()
    seconds = min(timer.repeat(repeat=5, number=loops))
    print(f"Benchmark{name} {loops} {seconds / loops * 1e9:.1f} ns/op")


def steady(codec: Codec, frame: sy.Frame) -> bytes:
    """Returns frame encoded after the first frame of its state, which codec has
    decoded, so a benchmark measures the frame that a stream sends most."""
    codec.decode(codec.encode(frame))
    return codec.encode(frame)


def shape(channels: int, samples: int) -> tuple[Codec, sy.Frame]:
    """Returns a codec of float32 channels and a frame of one series for each."""
    keys = list(range(1, channels + 1))
    codec = Codec(keys=keys, data_types=[sy.DataType.FLOAT32] * channels)
    series = [sy.Series(np.arange(samples, dtype=np.float32)) for _ in keys]
    return codec, sy.Frame(channels=keys, series=series)


def bench_shape(channels: int, samples: int) -> None:
    """Benchmarks the steady encode and decode of one float32 series per channel."""
    codec, frame = shape(channels, samples)
    wire = steady(codec, frame)
    name = f"channels={channels}/samples={samples}"
    bench(f"Encode/{name}", lambda: codec.encode(frame))
    bench(f"Decode/{name}", lambda: codec.decode(wire))


def main() -> None:
    for channels, samples in [(1, 3), (100, 10), (1000, 1)]:
        bench_shape(channels, samples)
    # A read can return many series for one channel, such as one for each stored
    # domain.
    codec = Codec(keys=[1], data_types=[sy.DataType.FLOAT32])
    series = [sy.Series(np.arange(10, dtype=np.float32)) for _ in range(100)]
    wire = steady(codec, sy.Frame(channels=[1] * 100, series=series))
    bench("Decode/channels=1/series=100/samples=10", lambda: codec.decode(wire))


if __name__ == "__main__":
    main()
