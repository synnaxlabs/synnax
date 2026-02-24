#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Simulates an acoustic sensor on an electric aircraft propulsion motor. The motor
runs a repeating 60-second profile:

  0-20s   Ramp from idle (600 RPM) to cruise (18 000 RPM)
  20-40s  Hold at cruise
  40-55s  Ramp down to idle
  55-60s  Idle

The signal contains:
  - Shaft fundamental frequency (RPM / 60) and 5 harmonics
  - Blade-pass frequency (shaft × 7 blades) with 2 harmonics
  - Outer-race bearing defect tone (shaft × 3.58, BPFO for a typical bearing)
  - RPM-dependent broadband noise floor (louder at higher speeds)
  - Occasional transient impact events (simulating minor imbalance strikes)

Recommended spectrogram settings:
  Sample rate: 48000    FFT size: 4096    Overlap: 75%
  dB min: -60           dB max: 0         Freq max: 8000
  Colormap: inferno
"""

import numpy as np

import synnax as sy

# --- Motor profile -----------------------------------------------------------

SAMPLE_RATE = 48_000
CHUNK_DURATION_S = 0.1
CHUNK_SIZE = int(SAMPLE_RATE * CHUNK_DURATION_S)
SAMPLE_PERIOD_NS = int(1e9 / SAMPLE_RATE)

PROFILE_DURATION_S = 60.0
IDLE_RPM = 600.0
CRUISE_RPM = 18_000.0
NUM_BLADES = 7
BPFO_RATIO = 3.58  # ball-pass frequency, outer race


def rpm_at(t: np.ndarray) -> np.ndarray:
    """RPM as a function of time within the 60 s profile."""
    phase = t % PROFILE_DURATION_S
    rpm = np.full_like(phase, IDLE_RPM)

    # 0-20s: ramp up
    ramp_up = (phase >= 0) & (phase < 20)
    rpm[ramp_up] = IDLE_RPM + (CRUISE_RPM - IDLE_RPM) * (phase[ramp_up] / 20)

    # 20-40s: cruise
    cruise = (phase >= 20) & (phase < 40)
    rpm[cruise] = CRUISE_RPM

    # 40-55s: ramp down
    ramp_down = (phase >= 40) & (phase < 55)
    rpm[ramp_down] = CRUISE_RPM - (CRUISE_RPM - IDLE_RPM) * (
        (phase[ramp_down] - 40) / 15
    )

    # 55-60s: idle
    return rpm


def generate_chunk(t: np.ndarray, phase_acc: float) -> tuple[np.ndarray, float]:
    """Generate one chunk of motor acoustic data.

    Returns the signal and the updated phase accumulator (for continuous phase
    across chunk boundaries).
    """
    rpm = rpm_at(t)
    shaft_freq = rpm / 60.0  # Hz

    # Accumulate phase continuously to avoid clicks at chunk boundaries.
    dt = 1.0 / SAMPLE_RATE
    inst_phase = np.cumsum(shaft_freq) * dt * 2 * np.pi
    inst_phase += phase_acc
    new_phase_acc = float(inst_phase[-1]) % (2 * np.pi)

    signal = np.zeros_like(t)

    # Shaft harmonics (1× through 6×), decreasing amplitude.
    for h in range(1, 7):
        amp = 1.0 / h
        signal += amp * np.sin(h * inst_phase)

    # Blade-pass frequency (shaft × NUM_BLADES) and two harmonics.
    for h in range(1, 3):
        amp = 0.6 / h
        signal += amp * np.sin(h * NUM_BLADES * inst_phase)

    # Bearing defect tone — narrow, quieter.
    signal += 0.15 * np.sin(BPFO_RATIO * inst_phase)
    signal += 0.08 * np.sin(2 * BPFO_RATIO * inst_phase)

    # RPM-dependent broadband noise floor.
    rpm_frac = (rpm - IDLE_RPM) / (CRUISE_RPM - IDLE_RPM)
    noise_level = 0.02 + 0.15 * rpm_frac
    signal += noise_level * np.random.randn(len(t))

    # Occasional transient impacts — short broadband bursts every ~4 seconds.
    phase_in_profile = t % PROFILE_DURATION_S
    impact_pos = phase_in_profile % 4.0
    impact_mask = impact_pos < 0.008  # 8 ms burst
    signal += impact_mask * np.random.randn(len(t)) * (1.0 + 2.0 * rpm_frac)

    return signal, new_phase_acc


# --- Synnax streaming ---------------------------------------------------------

client = sy.Synnax()

time_ch = client.channels.create(
    name="spectrogram_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

data_ch = client.channels.create(
    name="spectrogram_signal",
    index=time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

print("Electric propulsion motor acoustic simulator")
print(f"  Profile: {IDLE_RPM:.0f} → {CRUISE_RPM:.0f} RPM, {PROFILE_DURATION_S:.0f}s cycle")
print(f"  Streaming {SAMPLE_RATE} Hz in {CHUNK_SIZE}-sample chunks")
print(f"  time: {time_ch.key} ({time_ch.name})")
print(f"  data: {data_ch.key} ({data_ch.name})")
print("Press Ctrl+C to stop.\n")

start = sy.TimeStamp.now()
loop = sy.Loop(sy.Rate.HZ * (1 / CHUNK_DURATION_S))
elapsed_samples = 0
phase_acc = 0.0

with client.open_writer(start, [time_ch.key, data_ch.key]) as writer:
    while loop.wait():
        t = (elapsed_samples + np.arange(CHUNK_SIZE)) / SAMPLE_RATE

        signal, phase_acc = generate_chunk(t, phase_acc)

        chunk_start_ns = int(start) + elapsed_samples * SAMPLE_PERIOD_NS
        timestamps = chunk_start_ns + np.arange(CHUNK_SIZE) * SAMPLE_PERIOD_NS

        writer.write(
            {
                time_ch.key: timestamps,
                data_ch.key: signal.astype(np.float32),
            }
        )

        elapsed_samples += CHUNK_SIZE
        elapsed_s = elapsed_samples / SAMPLE_RATE
        if elapsed_samples % (SAMPLE_RATE * 5) < CHUNK_SIZE:
            rpm_now = rpm_at(np.array([elapsed_s]))[0]
            print(f"  {elapsed_s:5.0f}s  {rpm_now:7.0f} RPM  {elapsed_samples:>12,} samples")
