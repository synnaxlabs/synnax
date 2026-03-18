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
    writer_keys: list[int]

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

        self.writer_keys = [ts.key]
        for label, dt in [
            ("sensor", sy.DataType.FLOAT32),
            ("reference", sy.DataType.FLOAT32),
            ("precision", sy.DataType.FLOAT64),
        ]:
            name = f"{label}_{self.suffix}"
            self.raw[label] = name
            ch = self.client.channels.create(name=name, data_type=dt, index=ts.key)
            self.writer_keys.append(ch.key)
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
                name=name,
                expression=expr,
            )
            assert error is None, f"Failed to create {name}: {error}"

    def _write_raw(self, writer: sy.Writer) -> None:
        writer.write(
            {
                self.ts_key: sy.TimeStamp.now(),
                self.client.channels.retrieve(self.raw["sensor"]).key: np.array(
                    [150.0], dtype=np.float32
                ),
                self.client.channels.retrieve(self.raw["reference"]).key: np.array(
                    [50.0], dtype=np.float32
                ),
                self.client.channels.retrieve(self.raw["precision"]).key: np.array(
                    [150.0], dtype=np.float64
                ),
            }
        )

    def _read_calc(
        self, streamer: sy.Streamer, calc_label: str, timeout: float = 5
    ) -> float | None:
        calc_ch = self.client.channels.retrieve(self.calcs[calc_label])
        result = streamer.read(timeout=timeout)
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

        calc_keys = [
            self.client.channels.retrieve(self.calcs[label]).key
            for label in ["scaled", "monitor", "combined"]
        ]
        with self.client.open_streamer(calc_keys) as streamer:
            start = sy.TimeStamp.now()
            with self.client.open_writer(start, self.writer_keys) as writer:
                self._write_raw(writer)

            for _ in range(3):
                result = streamer.read(timeout=5)
                if result is None:
                    continue
                scaled_ch = self.client.channels.retrieve(self.calcs["scaled"])
                if len(result[scaled_ch.key]) > 0:
                    val = float(result[scaled_ch.key][0])
                    assert abs(val - 300.0) < 1.0, f"scaled: expected 300, got {val}"
                    break

        self.log("Phase 1: All calcs verified")

    def test_phase2_trigger_cascade(self) -> None:
        self.log("Phase 2: Editing scaled to change DataType from f32 to f64")

        self.console.notifications.close_all()

        scaled_ch = self.client.channels.retrieve(self.calcs["scaled"])
        scaled_ch.expression = f"return {self.raw['precision']} * 2"
        self.client.channels.create(scaled_ch)

        for _ in range(20):
            dt = self._get_data_type("scaled")
            if dt == sy.DataType.FLOAT64:
                break
            sy.sleep(0.25)

        assert (
            self._get_data_type("scaled") == sy.DataType.FLOAT64
        ), f"scaled should be f64 after edit, got {self._get_data_type('scaled')}"

        self.log("Phase 2: Cascade triggered")

    def test_phase3_verify_compatible_cascade(self) -> None:
        self.log("Phase 3: Verifying compatible cascade (monitor)")

        assert (
            self._get_data_type("monitor") == sy.DataType.FLOAT64
        ), f"monitor should be f64 after cascade, got {self._get_data_type('monitor')}"

        monitor_ch = self.client.channels.retrieve(self.calcs["monitor"])
        with self.client.open_streamer(monitor_ch.key) as streamer:
            start = sy.TimeStamp.now()
            with self.client.open_writer(start, self.writer_keys) as writer:
                for _ in range(5):
                    self._write_raw(writer)
                    result = streamer.read(timeout=5)
                    if result is not None and len(result[monitor_ch.key]) > 0:
                        val = float(result[monitor_ch.key][0])
                        if val != 0.0:
                            assert (
                                abs(val - 3000.0) < 1.0
                            ), f"monitor: expected 3000, got {val}"
                            break
                    sy.sleep(0.5)

        self.log("Phase 3: Compatible cascade verified")

    def test_phase4_verify_incompatible_cascade(self) -> None:
        self.log("Phase 4: Verifying incompatible cascade (combined)")

        assert self._get_data_type("combined") == sy.DataType.FLOAT32, (
            f"combined should stay f32 after failed cascade, "
            f"got {self._get_data_type('combined')}"
        )

        notifs = self.console.notifications.check(timeout=3 * sy.TimeSpan.SECOND)
        error_notifs = [
            n
            for n in notifs
            if n.get("type") == "error" and "Connected" not in n.get("message", "")
        ]
        self.log(f"Phase 4: Error notifications: {error_notifs}")

        self.log("Phase 4: Incompatible cascade verified")

    def test_phase5_recovery(self) -> None:
        self.log("Phase 5: Fixing combined via Console")

        self.console.notifications.close_all()

        combined_ch = self.client.channels.retrieve(self.calcs["combined"])
        combined_ch.expression = (
            f"return f32({self.calcs['scaled']}) + {self.raw['reference']}"
        )
        self.client.channels.create(combined_ch)
        sy.sleep(2)

        notifs = self.console.notifications.check()
        error_notifs = [
            n
            for n in notifs
            if n.get("type") == "error" and "Connected" not in n.get("message", "")
        ]
        assert (
            len(error_notifs) == 0
        ), f"Expected no errors after fix, got: {error_notifs}"

        assert (
            self._get_data_type("combined") == sy.DataType.FLOAT32
        ), f"combined should be f32 after fix, got {self._get_data_type('combined')}"

        self.log("Phase 5: Recovery verified")

    def teardown(self) -> None:
        calc_names = list(reversed(list(self.calcs.values())))
        calc_keys = []
        for name in calc_names:
            try:
                ch = self.client.channels.retrieve(name)
                calc_keys.append(ch.key)
                if ch.index != 0:
                    calc_keys.append(ch.index)
            except Exception:
                pass
        if calc_keys:
            self.client.channels.delete(calc_keys)
        raw_names = list(self.raw.values()) + [f"ts_{self.suffix}"]
        self.client.channels.delete(raw_names)
        super().teardown()
