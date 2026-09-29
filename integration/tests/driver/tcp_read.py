#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
from examples.simulators.scpi import INITIAL_VOLTAGE, STATUS, SCPISim, reading

import synnax as sy
from synnax import bus, library, tcp
from tests.driver.bus_task import (
    Bound,
    BusCase,
    Collector,
    assert_equal,
    assert_increasing,
    assert_indexed,
)

VOLTAGE_SCALE = 2.0
VOLTAGE_OFFSET = -0.5
CURRENT_SCALE = 1000.0
KELVIN_OFFSET = 273.15
PRESSURE_SCALE = 1000.0


def reading_entry() -> library.MessageEntry:
    return library.MessageEntry(
        name="reading",
        format="text",
        query="READ?",
        fields=[
            library.DelimitedField(name="count", position=0),
            library.DelimitedField(
                name="voltage", position=1, scale=VOLTAGE_SCALE, offset=VOLTAGE_OFFSET
            ),
            library.DelimitedField(
                name="current", position=2, scale=CURRENT_SCALE, units="mA"
            ),
        ],
    )


class TCPRead(BusCase):
    """Polls three SCPI queries and checks each decoded reply.

    READ? replies carry delimited fields with scale and offset, STAT? replies carry
    tagged fields, and MEAS:VOLT? replies carry one bare number. Each query is its own
    message with its own index.
    """

    sim_classes = [SCPISim]
    prefix = "tcp_read"
    reading: Bound
    status: Bound
    output: Bound

    def create_entries(self) -> list[library.Entry]:
        return [
            reading_entry(),
            library.MessageEntry(
                name="status",
                format="text",
                query="STAT?",
                fields=[
                    library.TaggedField(
                        name="temperature", tag="T=", offset=KELVIN_OFFSET, units="K"
                    ),
                    library.TaggedField(
                        name="pressure", tag="P=", scale=PRESSURE_SCALE, units="Pa"
                    ),
                ],
            ),
            library.MessageEntry(
                name="output",
                format="text",
                query="MEAS:VOLT?",
                fields=[library.DelimitedField(name="voltage", position=0)],
            ),
        ]

    def create_task(self, device: sy.Device, library_key: library.Key) -> tcp.ReadTask:
        reading_msg, self.reading = self.bind_read(
            "reading",
            ["count", "voltage", "current"],
            {"count": sy.DataType.UINT32},
        )
        status_msg, self.status = self.bind_read("status", ["temperature", "pressure"])
        output_msg, self.output = self.bind_read("output", ["voltage"])
        return tcp.ReadTask(
            name="TCP Read",
            device=device.key,
            library=library_key,
            messages=[reading_msg, status_msg, output_msg],
            rate=20 * sy.Rate.HZ,
            timeout=500 * sy.TimeSpan.MILLISECOND,
            framing=bus.DelimiterFraming(delimiter="\n"),
        )

    def run(self) -> None:
        bounds = [self.reading, self.status, self.output]
        with self.statuses() as statuses:
            with self.collect(bounds) as samples, self.task.run():
                samples.wait(count=20)
            self.assert_no_problems(statuses)
        for name, bound in zip(["reading", "status", "output"], bounds):
            assert_indexed(name, bound, samples)
        self._verify_reading(samples, self.reading)
        self._verify_status(samples, self.status)
        assert_equal(
            "output.voltage",
            samples[self.output.fields["voltage"]],
            [INITIAL_VOLTAGE] * samples.count(self.output),
        )

    @staticmethod
    def _verify_reading(samples: Collector, bound: Bound) -> None:
        counts = samples[bound.fields["count"]].astype(np.int64)
        # One query in flight at a time, so every reply the simulator sent arrives.
        assert_increasing("reading.count", counts, step=1)
        replies = [reading(int(n)) for n in counts]
        assert_equal(
            "reading.voltage",
            samples[bound.fields["voltage"]],
            [v * VOLTAGE_SCALE + VOLTAGE_OFFSET for v, _ in replies],
        )
        current = samples[bound.fields["current"]]
        assert_equal(
            "reading.current", current, [c * CURRENT_SCALE for _, c in replies]
        )
        if not np.any(current < 0):
            raise AssertionError("reading.current: expected negative samples")

    @staticmethod
    def _verify_status(samples: Collector, bound: Bound) -> None:
        n = samples.count(bound)
        temperature, pressure = STATUS
        assert_equal(
            "status.temperature",
            samples[bound.fields["temperature"]],
            [temperature + KELVIN_OFFSET] * n,
        )
        assert_equal(
            "status.pressure",
            samples[bound.fields["pressure"]],
            [pressure * PRESSURE_SCALE] * n,
        )


class TCPReadTimeout(BusCase):
    """Mutes the instrument while the task polls it. The task warns once a reply
    times out, invents no samples while the instrument is silent, and resumes when it
    answers again."""

    sim_classes = [SCPISim]
    prefix = "tcp_read_timeout"
    reading: Bound

    def create_entries(self) -> list[library.Entry]:
        return [reading_entry()]

    def create_task(self, device: sy.Device, library_key: library.Key) -> tcp.ReadTask:
        reading_msg, self.reading = self.bind_read(
            "reading", ["count"], {"count": sy.DataType.UINT32}
        )
        return tcp.ReadTask(
            name="TCP Read Timeout",
            device=device.key,
            library=library_key,
            messages=[reading_msg],
            rate=10 * sy.Rate.HZ,
            timeout=200 * sy.TimeSpan.MILLISECOND,
            framing=bus.DelimiterFraming(delimiter="\n"),
        )

    def run(self) -> None:
        sim = self.simulator(SCPISim)
        with self.statuses() as statuses:
            with self.collect([self.reading]) as samples, self.task.run():
                samples.wait(count=5)
                self.assert_no_problems(statuses)
                sim.mute()
                message = self.wait_for_status(statuses, "warning")
                self.log(f"Warning while muted: {message}")
                samples.drain()
                before = samples.count(self.reading)
                sim.unmute()
                samples.wait(count=before + 5)
        assert_indexed("reading", self.reading, samples)
        # The muted simulator counts no replies, so a sample invented during the
        # silence would break the run of counts.
        assert_increasing(
            "reading.count", samples[self.reading.fields["count"]], step=1
        )
