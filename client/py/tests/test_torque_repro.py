#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import numpy as np
import pytest

import synnax as sy
from synnax.util.random import random_name


@pytest.mark.calculations
class TestTorqueReproduction:
    """Reproduce the Torque calculated channel failure.

    The user's sequence of events:
    1. Created the full calc chain with original expressions (all worked)
    2. Updated input_power's expression from
       `return i64(f64(Heat_Rejection)+Terminal_Power)` (infers i64)
       to `return Heat_Rejection+f32(Terminal_Power)` (infers f32)
    3. The stored DataType for input_power stayed i64 (stale)
    4. Torque references input_power, so its calculator sees input_power as i64
    5. Torque's expression `return f32(input_power*60)/(2*(3.14159)*(drive_speed_fb))`
       compiles with input_power as i64 param, everything works...
       BUT the calculation service recompiles input_power's calculator with
       DataType=i64 and expression that returns f32 -> the Wasm function signature
       says i64 output but the body produces f32
    """

    def test_stale_datatype_breaks_downstream(self, client: sy.Synnax):
        """Exact reproduction of what the user did.

        Step 1: Create full chain with original expressions (works fine)
        Step 2: Update input_power expression (DataType goes stale)
        Step 3: Torque should break because input_power's calculator is broken
        """
        ts = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )

        # --- Raw channels ---
        drive_speed_fb = client.channels.create(
            name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32,
        )
        flow = client.channels.create(
            name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32,
        )
        t_inlet = client.channels.create(
            name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32,
        )
        t_outlet = client.channels.create(
            name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32,
        )
        por = client.channels.create(
            name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT64,
        )
        load_current = client.channels.create(
            name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT64,
        )

        # --- Layer 1 calcs (original expressions) ---
        delta_temp = client.channels.create(
            name=random_name(),
            expression=(
                f"inlet_c := f32(({t_inlet.name}-32)/1.8)\n"
                f"outlet_c := f32(({t_outlet.name}-32)/1.8)\n"
                f"return f32(outlet_c - inlet_c)"
            ),
        )
        mass_flow_rate = client.channels.create(
            name=random_name(),
            expression=f"return {flow.name}*0.058",
        )
        specific_heat = client.channels.create(
            name=random_name(),
            expression=(
                f"return 1642+(4.937)*"
                f"(((({t_inlet.name}-32)/1.8)+(({t_outlet.name}-32)/1.8))/2)"
            ),
        )
        terminal_power = client.channels.create(
            name=random_name(),
            expression=f"return (({load_current.name})*({por.name}))/1000",
        )

        # --- Layer 2 ---
        heat_rejection = client.channels.create(
            name=random_name(),
            expression=(
                f"return ({delta_temp.name}*{mass_flow_rate.name}"
                f"*{specific_heat.name})/1000.0"
            ),
        )

        # --- Layer 3: input_power with ORIGINAL expression (infers i64) ---
        input_power = client.channels.create(
            name=random_name(),
            expression=(
                f"return i64(f64({heat_rejection.name})+{terminal_power.name})"
            ),
        )
        assert input_power.data_type == sy.DataType.INT64

        # --- Layer 4: Torque with ORIGINAL expression ---
        torque = client.channels.create(
            name=random_name(),
            expression=(
                f"return f32({input_power.name}*60)"
                f"/(2*(3.14159)*({drive_speed_fb.name}))"
            ),
        )
        assert torque.data_type == sy.DataType.FLOAT32

        # --- Verify original chain works ---
        start = sy.TimeStamp.now()
        with client.open_streamer(torque.key) as streamer:
            with client.open_writer(
                start,
                [
                    ts.key, drive_speed_fb.key, flow.key,
                    t_inlet.key, t_outlet.key, por.key, load_current.key,
                ],
            ) as writer:
                writer.write({
                    ts.key: sy.TimeStamp.now(),
                    drive_speed_fb.key: np.array([1000.0], dtype=np.float32),
                    flow.key: np.array([100.0], dtype=np.float32),
                    t_inlet.key: np.array([150.0], dtype=np.float32),
                    t_outlet.key: np.array([200.0], dtype=np.float32),
                    por.key: np.array([480.0], dtype=np.float64),
                    load_current.key: np.array([100.0], dtype=np.float64),
                })
                frame = streamer.read(timeout=10)
                assert frame is not None, "Original chain should work"

        # --- NOW: Update input_power expression (like the user did) ---
        # Original: return i64(f64(Heat_Rejection)+Terminal_Power) -> inferred i64
        # New:      return Heat_Rejection+f32(Terminal_Power)      -> should be f32
        input_power.expression = (
            f"return {heat_rejection.name}+f32({terminal_power.name})"
        )
        client.channels.create(input_power)

        # Check: is the stored DataType stale?
        retrieved = client.channels.retrieve(input_power.key)
        print(f"input_power stored DataType: {retrieved.data_type}")
        print(f"input_power expression: {retrieved.expression}")

        # Give the server time to recompile calculators
        time.sleep(2)

        # --- Try to stream Torque after the update ---
        start = sy.TimeStamp.now()
        with client.open_streamer(torque.key) as streamer:
            with client.open_writer(
                start,
                [
                    ts.key, drive_speed_fb.key, flow.key,
                    t_inlet.key, t_outlet.key, por.key, load_current.key,
                ],
            ) as writer:
                writer.write({
                    ts.key: sy.TimeStamp.now(),
                    drive_speed_fb.key: np.array([1000.0], dtype=np.float32),
                    flow.key: np.array([100.0], dtype=np.float32),
                    t_inlet.key: np.array([150.0], dtype=np.float32),
                    t_outlet.key: np.array([200.0], dtype=np.float32),
                    por.key: np.array([480.0], dtype=np.float64),
                    load_current.key: np.array([100.0], dtype=np.float64),
                })
                frame = streamer.read(timeout=10)
                assert frame is not None, "Chain should still work after expression update"
                assert len(frame[torque.key]) > 0
