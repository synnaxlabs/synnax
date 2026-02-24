#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np

import synnax as sy
from examples.simulators.simdaq import SimDAQ

SAMPLE_RATE = 48_000
LOOP_RATE = 50
SAMPLES_PER_TICK = SAMPLE_RATE // LOOP_RATE  # 960
SAMPLE_PERIOD_NS = int(1e9 / SAMPLE_RATE)

NUM_BLADES = 7
BPFO_RATIO = 3.58


def generate_acoustic(
    rpm: float,
    elapsed_samples: int,
    phase_acc: float,
) -> tuple[np.ndarray, float]:
    """Generate one tick of motor acoustic data (960 samples at 48 kHz).

    Returns the signal and updated phase accumulator for continuity across chunks.
    """
    shaft_freq = rpm / 60.0
    dt = 1.0 / SAMPLE_RATE
    n = SAMPLES_PER_TICK

    inst_phase = np.full(n, shaft_freq)
    inst_phase = np.cumsum(inst_phase) * dt * 2 * np.pi
    inst_phase += phase_acc
    new_phase_acc = float(inst_phase[-1]) % (2 * np.pi)

    signal = np.zeros(n, dtype=np.float64)

    for h in range(1, 7):
        signal += (1.0 / h) * np.sin(h * inst_phase)

    for h in range(1, 3):
        signal += (0.6 / h) * np.sin(h * NUM_BLADES * inst_phase)

    signal += 0.15 * np.sin(BPFO_RATIO * inst_phase)
    signal += 0.08 * np.sin(2 * BPFO_RATIO * inst_phase)

    cruise_rpm = 18_000.0
    idle_rpm = 0.0
    rpm_range = cruise_rpm - idle_rpm
    rpm_frac = np.clip((rpm - idle_rpm) / rpm_range, 0, 1) if rpm_range > 0 else 0.0
    noise_level = 0.02 + 0.15 * rpm_frac
    signal += noise_level * np.random.randn(n)

    t_in_profile = (elapsed_samples + np.arange(n)) / SAMPLE_RATE
    impact_pos = t_in_profile % 4.0
    impact_mask = impact_pos < 0.008
    signal += impact_mask * np.random.randn(n) * (1.0 + 2.0 * rpm_frac)

    return signal.astype(np.float32), new_phase_acc


class MotorSimDAQ(SimDAQ):
    """Simulates an electric propulsion motor with RPM command input,
    control-rate sensor outputs, and a 48 kHz acoustic signal for
    spectrogram visualization.
    """

    description = "Run motor simulator standalone"

    def _create_channels(self) -> None:
        self.log("Creating channels...")
        client = self.client

        self.cmd_time = client.channels.create(
            name="motor_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="motor_rpm_cmd",
            index=self.cmd_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        self.daq_time = client.channels.create(
            name="motor_daq_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        for name in ["motor_rpm", "motor_current", "motor_temp", "motor_thrust"]:
            client.channels.create(
                name=name,
                index=self.daq_time.key,
                data_type=sy.DataType.FLOAT32,
                retrieve_if_name_exists=True,
            )

        self.acoustic_time = client.channels.create(
            name="motor_acoustic_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name="motor_acoustic",
            index=self.acoustic_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        self.log("Channels created successfully")

    def _run_loop(self) -> None:
        self.log("Starting simulation loop...")
        loop = sy.Loop(sy.Rate.HZ * LOOP_RATE)
        loop_count = 0

        rpm_cmd = 0.0
        rpm = 0.0
        temp = 25.0
        phase_acc = 0.0
        elapsed_samples = 0
        dt = 1.0 / LOOP_RATE

        with self.client.open_streamer(["motor_rpm_cmd"]) as streamer:
            start = sy.TimeStamp.now()
            with self.client.open_writer(
                start=start,
                channels=[
                    "motor_daq_time",
                    "motor_rpm",
                    "motor_current",
                    "motor_temp",
                    "motor_thrust",
                    "motor_acoustic_time",
                    "motor_acoustic",
                ],
                name="Motor Sim DAQ",
            ) as writer:
                while self._running and loop.wait():
                    while True:
                        frame = streamer.read(timeout=0)
                        if frame is not None:
                            data = frame.get("motor_rpm_cmd")
                            if len(data) > 0:
                                new_val = data[-1]
                                if hasattr(new_val, "item"):
                                    new_val = new_val.item()
                                if new_val != rpm_cmd:
                                    self.log(
                                        f"RPM cmd: {rpm_cmd:.0f} -> {new_val:.0f}"
                                    )
                                rpm_cmd = float(new_val)
                        else:
                            break

                    rpm += (rpm_cmd - rpm) * dt / 0.5
                    current = (
                        2.0
                        + (rpm / 18_000) * 45.0
                        + np.random.uniform(-0.1, 0.1)
                    )
                    steady_temp = 25.0 + (rpm / 18_000) * 60.0
                    temp += (steady_temp - temp) * dt / 15.0
                    temp += np.random.uniform(-0.05, 0.05)
                    thrust = (
                        4e-6 * rpm**2
                        + np.random.uniform(-0.5, 0.5)
                    )

                    signal, phase_acc = generate_acoustic(
                        rpm, elapsed_samples, phase_acc
                    )

                    now = sy.TimeStamp.now()
                    chunk_start_ns = (
                        int(start) + elapsed_samples * SAMPLE_PERIOD_NS
                    )
                    acoustic_timestamps = (
                        chunk_start_ns
                        + np.arange(SAMPLES_PER_TICK) * SAMPLE_PERIOD_NS
                    )

                    writer.write(
                        {
                            "motor_daq_time": now,
                            "motor_rpm": np.float32(rpm),
                            "motor_current": np.float32(current),
                            "motor_temp": np.float32(temp),
                            "motor_thrust": np.float32(thrust),
                            "motor_acoustic_time": acoustic_timestamps,
                            "motor_acoustic": signal,
                        }
                    )

                    elapsed_samples += SAMPLES_PER_TICK
                    loop_count += 1
                    if loop_count % (LOOP_RATE * 2) == 0:
                        self.log(
                            f"rpm={rpm:.0f}/{rpm_cmd:.0f}, "
                            f"I={current:.1f}A, "
                            f"T={temp:.1f}C, "
                            f"F={thrust:.1f}N"
                        )

        self.log("Simulation loop stopped")


if __name__ == "__main__":
    MotorSimDAQ.main()
