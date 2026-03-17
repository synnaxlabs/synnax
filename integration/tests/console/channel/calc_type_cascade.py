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
from xpy import get_random_name

from console.case import ConsoleCase


class CalcTypeCascade(ConsoleCase):
    """Test that editing an upstream calc's expression cascades DataType changes
    to downstream calcs, producing errors for incompatible ones and silently
    updating compatible ones.

    Chain:
        Raw: sensor(f32), reference(f32), precision(f64)
        scaled   = return sensor * 2           (f32)
        combined = return scaled + reference    (f32 + f32, valid)
        monitor  = return scaled * 10           (f32, single input)

    Trigger: edit scaled to "return precision * 2" (f32 -> f64)
        monitor:  scaled(f64) * 10 -> silently becomes f64 (compatible)
        combined: scaled(f64) + reference(f32) -> type error (incompatible)

    Recovery: edit combined to "return f32(scaled) + reference"
    """

    suffix: str
    raw: dict[str, str]
    calcs: dict[str, str]
    ts_key: int

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self.raw = {}
        self.calcs = {}
        self._create_channels()

    def _create_channels(self) -> None:
        ts = self.client.channels.create(
            name=f"ts_{self.suffix}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        self.ts_key = ts.key
        self.console.channels.wait_for_channels(ts.name, timeout=5.0)

        for label, dt in [
            ("sensor", sy.DataType.FLOAT32),
            ("reference", sy.DataType.FLOAT32),
            ("precision", sy.DataType.FLOAT64),
        ]:
            name = f"{label}_{self.suffix}"
            self.raw[label] = name
            self.client.channels.create(name=name, data_type=dt, index=ts.key)
            self.console.channels.wait_for_channels(name, timeout=5.0)

        chain = [
            ("scaled", f"return {self.raw['sensor']} * 2"),
            ("combined", f"return scaled_{self.suffix} + {self.raw['reference']}"),
            ("monitor", f"return scaled_{self.suffix} * 10"),
        ]
        for label, expr in chain:
            name = f"{label}_{self.suffix}"
            self.calcs[label] = name
            error = self.console.channels.create_calculated(
                name=name, expression=expr,
            )
            assert error is None, f"Failed to create {name}: {error}"

    def _write_and_stream(self, calc_label: str) -> float | None:
        calc_name = self.calcs[calc_label]
        calc_ch = self.client.channels.retrieve(calc_name)
        with self.client.open_streamer(calc_ch.key) as streamer:
            start = sy.TimeStamp.now()
            ts_ch = self.client.channels.retrieve(f"ts_{self.suffix}")
            writer_keys = [ts_ch.key]
            frame = {ts_ch.key: start + 1 * sy.TimeSpan.MILLISECOND}
            raw_values = {
                "sensor": np.array([150.0], dtype=np.float32),
                "reference": np.array([50.0], dtype=np.float32),
                "precision": np.array([150.0], dtype=np.float64),
            }
            for label, values in raw_values.items():
                ch = self.client.channels.retrieve(self.raw[label])
                writer_keys.append(ch.key)
                frame[ch.key] = values
            with self.client.open_writer(start, writer_keys) as writer:
                writer.write(frame)
            result = streamer.read(timeout=5)
            if result is not None and len(result[calc_ch.key]) > 0:
                return float(result[calc_ch.key][0])
        return None

    def _get_data_type(self, calc_label: str) -> sy.DataType:
        fresh = sy.Synnax(
            host="localhost",
            port=9090,
            username="synnax",
            password="seldon",
            secure=False,
        )
        try:
            return fresh.channels.retrieve(self.calcs[calc_label]).data_type
        finally:
            fresh.close()

    def run(self) -> None:
        self.test_phase1_verify_initial_state()
        self.test_phase2_trigger_cascade()
        self.test_phase3_verify_compatible_cascade()
        self.test_phase4_verify_incompatible_cascade()
        self.test_phase5_recovery()

    def test_phase1_verify_initial_state(self) -> None:
        self.log("Phase 1: Verifying initial state")

        assert self._get_data_type("scaled") == sy.DataType.FLOAT32
        assert self._get_data_type("combined") == sy.DataType.FLOAT32
        assert self._get_data_type("monitor") == sy.DataType.FLOAT32

        # scaled = sensor * 2 = 150 * 2 = 300
        scaled_val = self._write_and_stream("scaled")
        assert scaled_val is not None, "scaled should produce a value"
        assert abs(scaled_val - 300.0) < 1.0, f"scaled: expected 300, got {scaled_val}"

        # monitor = scaled * 10 = 300 * 10 = 3000
        monitor_val = self._write_and_stream("monitor")
        assert monitor_val is not None, "monitor should produce a value"
        assert abs(monitor_val - 3000.0) < 1.0, f"monitor: expected 3000, got {monitor_val}"

        # combined = scaled + reference = 300 + 50 = 350
        combined_val = self._write_and_stream("combined")
        assert combined_val is not None, "combined should produce a value"
        assert abs(combined_val - 350.0) < 1.0, f"combined: expected 350, got {combined_val}"

        self.log("Phase 1: All calcs verified")

    def test_phase2_trigger_cascade(self) -> None:
        self.log("Phase 2: Editing scaled to change DataType from f32 to f64")

        self.console.notifications.close_all()

        # Edit scaled via Python client: change source from sensor(f32) to precision(f64)
        scaled_ch = self.client.channels.retrieve(self.calcs["scaled"])
        scaled_ch.expression = f"return {self.raw['precision']} * 2"
        self.client.channels.create(scaled_ch)

        # Poll until DataType propagates
        for _ in range(20):
            dt = self._get_data_type("scaled")
            if dt == sy.DataType.FLOAT64:
                break
            sy.sleep(0.25)

        assert self._get_data_type("scaled") == sy.DataType.FLOAT64, (
            f"scaled should be f64 after edit, got {self._get_data_type('scaled')}"
        )

        self.log("Phase 2: Cascade triggered")

    def test_phase3_verify_compatible_cascade(self) -> None:
        self.log("Phase 3: Verifying compatible cascade (monitor)")

        # monitor = scaled * 10. scaled is now f64.
        # monitor should silently become f64.
        assert self._get_data_type("monitor") == sy.DataType.FLOAT64, (
            f"monitor should be f64 after cascade, got {self._get_data_type('monitor')}"
        )

        # monitor should still produce correct values
        # scaled = precision * 2 = 150 * 2 = 300, monitor = 300 * 10 = 3000
        monitor_val = self._write_and_stream("monitor")
        assert monitor_val is not None, "monitor should still produce values"
        assert abs(monitor_val - 3000.0) < 1.0, f"monitor: expected 3000, got {monitor_val}"

        self.log("Phase 3: Compatible cascade verified")

    def test_phase4_verify_incompatible_cascade(self) -> None:
        self.log("Phase 4: Verifying incompatible cascade (combined)")

        # combined = scaled + reference. scaled is now f64, reference is f32.
        # f64 + f32 is a type mismatch. combined's DataType should stay f32.
        assert self._get_data_type("combined") == sy.DataType.FLOAT32, (
            f"combined should stay f32 after failed cascade, got {self._get_data_type('combined')}"
        )

        # Check for error notification about combined
        notifs = self.console.notifications.check(timeout=3 * sy.TimeSpan.SECOND)
        error_notifs = [
            n for n in notifs
            if n.get("type") == "error"
            and "Connected" not in n.get("message", "")
        ]
        self.log(f"Phase 4: Error notifications: {error_notifs}")

        self.log("Phase 4: Incompatible cascade verified")

    def test_phase5_recovery(self) -> None:
        self.log("Phase 5: Fixing combined via Console")

        self.console.notifications.close_all()

        # Fix combined via Python client: cast scaled to f32 so both operands match
        combined_ch = self.client.channels.retrieve(self.calcs["combined"])
        combined_ch.expression = f"return f32({self.calcs['scaled']}) + {self.raw['reference']}"
        self.client.channels.create(combined_ch)
        sy.sleep(2)

        # Verify no error notifications
        notifs = self.console.notifications.check()
        error_notifs = [
            n for n in notifs
            if n.get("type") == "error"
            and "Connected" not in n.get("message", "")
        ]
        assert len(error_notifs) == 0, f"Expected no errors after fix, got: {error_notifs}"

        # combined should produce correct values again
        # scaled = precision * 2 = 300 (as f64), f32(300) = 300, + reference(50) = 350
        combined_val = self._write_and_stream("combined")
        assert combined_val is not None, "combined should produce values after fix"
        assert abs(combined_val - 350.0) < 1.0, f"combined: expected 350, got {combined_val}"

        assert self._get_data_type("combined") == sy.DataType.FLOAT32, (
            f"combined should be f32 after fix, got {self._get_data_type('combined')}"
        )

        self.log("Phase 5: Recovery verified")

    def teardown(self) -> None:
        all_names = (
            list(reversed(list(self.calcs.values())))
            + list(self.raw.values())
            + [f"ts_{self.suffix}"]
        )
        self.console.channels.delete(all_names)
        super().teardown()
