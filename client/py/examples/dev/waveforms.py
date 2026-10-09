#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Streams a few test-bench waveforms at 2 kHz for trying the line plot's spectrum mode,
triggered window, and measurements:

- waveform_sine: a 60 Hz sine of amplitude 5 with a 180 Hz harmonic of amplitude 1.
- waveform_noisy: the same 60 Hz sine with a 1 kHz tone and white noise on top.
- waveform_square: a 20 Hz square wave between 0 and 10 with a short rise.
- waveform_sweep: a sine whose frequency sweeps from 10 Hz to 400 Hz every 10 s.

Run it against a local Core and leave it running while you plot.
"""

import time

import numpy as np

import synnax as sy

RATE = 2000
CHUNK = 100  # samples per write, so 20 writes per second
SWEEP_PERIOD = 10.0  # s

client = sy.Synnax(host="localhost", port=9090, username="synnax", password="seldon")

index = client.channels.create(
    name="waveform_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)
names = ["waveform_sine", "waveform_noisy", "waveform_square", "waveform_sweep"]
keys = {
    name: client.channels.create(
        name=name,
        index=index.key,
        data_type=sy.DataType.FLOAT32,
        retrieve_if_name_exists=True,
    ).key
    for name in names
}

period = sy.TimeSpan.SECOND / RATE
start = sy.TimeStamp.now()
with client.open_writer(
    start=start,
    channels=["waveform_time", *names],
    enable_auto_commit=True,
) as writer:
    i = 0
    while True:
        n = np.arange(i, i + CHUNK)
        t = n / RATE
        stamps = start + (n * period).astype(np.int64)
        sine = 5 * np.sin(2 * np.pi * 60 * t) + np.sin(2 * np.pi * 180 * t)
        noisy = sine + 0.5 * np.sin(2 * np.pi * 1000 * t) + np.random.randn(CHUNK) * 0.3
        phase = (t * 20) % 1.0
        square = np.clip(phase * 50, 0, 1) * (phase < 0.5) * 10
        sweep_phase = (t % SWEEP_PERIOD) / SWEEP_PERIOD
        sweep = 3 * np.sin(2 * np.pi * (10 + 390 * sweep_phase) * t)
        writer.write(
            {
                index.key: stamps,
                keys["waveform_sine"]: sine.astype(np.float32),
                keys["waveform_noisy"]: noisy.astype(np.float32),
                keys["waveform_square"]: square.astype(np.float32),
                keys["waveform_sweep"]: sweep.astype(np.float32),
            }
        )
        i += CHUNK
        # Pace the writer to the wall clock so the stream stays live.
        ahead = (start + (i * period)) - sy.TimeStamp.now()
        if ahead > 0:
            time.sleep(sy.TimeSpan(ahead).seconds)
