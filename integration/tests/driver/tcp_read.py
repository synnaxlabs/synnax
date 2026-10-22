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
    assert_no_new_samples,
)

# Every reply arrives past POLL_TIMEOUT, yet before the task's quiet period after the
# miss ends, so the task must take it for late.
POLL_TIMEOUT = 400 * sy.TimeSpan.MILLISECOND
LATE_REPLY = 0.5
VOLTAGE_SCALE = 2.0
VOLTAGE_OFFSET = -0.5
CURRENT_SCALE = 1000.0
KELVIN_OFFSET = 273.15
PRESSURE_SCALE = 1000.0


def reading_entry() -> library.MessageEntry:
    return library.MessageEntry(
        name="reading",
        query="READ?",
        payload=library.TextPayload(
            fields=[
                library.DelimitedTextField(name="count", position=0),
                library.DelimitedTextField(
                    name="voltage",
                    position=1,
                    scale=VOLTAGE_SCALE,
                    offset=VOLTAGE_OFFSET,
                ),
                library.DelimitedTextField(
                    name="current", position=2, scale=CURRENT_SCALE, units="mA"
                ),
            ]
        ),
    )


def output_entry() -> library.MessageEntry:
    return library.MessageEntry(
        name="output",
        query="MEAS:VOLT?",
        payload=library.TextPayload(
            fields=[library.DelimitedTextField(name="voltage", position=0)]
        ),
    )


def verify_reading(samples: Collector, bound: Bound, step: int | None = None) -> None:
    """Asserts that the counts of the READ? replies increase, by step when given, and
    that each reply's voltage and current match its count."""
    counts = samples[bound.fields["count"]].astype(np.int64)
    assert_increasing("reading.count", counts, step=step)
    replies = [reading(int(n)) for n in counts]
    assert_equal(
        "reading.voltage",
        samples[bound.fields["voltage"]],
        [v * VOLTAGE_SCALE + VOLTAGE_OFFSET for v, _ in replies],
    )
    current = samples[bound.fields["current"]]
    assert_equal("reading.current", current, [c * CURRENT_SCALE for _, c in replies])
    if not np.any(current < 0):
        raise AssertionError("reading.current: expected negative samples")


def verify_output(samples: Collector, bound: Bound) -> None:
    """Asserts that every MEAS:VOLT? reply holds the initial voltage."""
    assert_equal(
        "output.voltage",
        samples[bound.fields["voltage"]],
        [INITIAL_VOLTAGE] * samples.count(bound),
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
                query="STAT?",
                payload=library.TextPayload(
                    fields=[
                        library.TaggedTextField(
                            name="temperature",
                            tag="T=",
                            offset=KELVIN_OFFSET,
                            units="K",
                        ),
                        library.TaggedTextField(
                            name="pressure", tag="P=", scale=PRESSURE_SCALE, units="Pa"
                        ),
                    ]
                ),
            ),
            output_entry(),
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
        for bound in bounds:
            assert_indexed(bound.message, bound, samples)
        # One query in flight at a time, so every reply the simulator sent arrives.
        verify_reading(samples, self.reading, step=1)
        self._verify_status(samples, self.status)
        verify_output(samples, self.output)

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


class TCPReadLateReply(BusCase):
    """Delays every instrument reply past the task's timeout. The task warns that it
    discarded a late reply, stores no late reply, not even as the answer to the next
    query, and reads again once replies are on time.

    A READ? reply stored as the MEAS:VOLT? answer would put a count in output.voltage.
    """

    sim_classes = [SCPISim]
    prefix = "tcp_read_late"
    reading: Bound
    output: Bound

    def create_entries(self) -> list[library.Entry]:
        return [reading_entry(), output_entry()]

    def create_task(self, device: sy.Device, library_key: library.Key) -> tcp.ReadTask:
        reading_msg, self.reading = self.bind_read(
            "reading",
            ["count", "voltage", "current"],
            {"count": sy.DataType.UINT32},
        )
        output_msg, self.output = self.bind_read("output", ["voltage"])
        return tcp.ReadTask(
            name="TCP Read Late Reply",
            device=device.key,
            library=library_key,
            messages=[reading_msg, output_msg],
            rate=10 * sy.Rate.HZ,
            timeout=POLL_TIMEOUT,
            framing=bus.DelimiterFraming(delimiter="\n"),
        )

    def run(self) -> None:
        sim = self.simulator(SCPISim)
        with self.statuses() as statuses:
            with self.collect(self.bounds) as samples, self.task.run():
                samples.wait(count=5)
                self.assert_no_problems(statuses)
                sim.delay(LATE_REPLY)
                message = self.wait_for_status(
                    statuses, "warning", containing="late reply to"
                )
                self.log(f"Warning while replies are late: {message}")
                assert_no_new_samples(samples, self.bounds)
                sim.delay(0)
                samples.wait_for_more(count=5)
                self.wait_for_status(statuses, "success")
        for bound in self.bounds:
            assert_indexed(bound.message, bound, samples)
        verify_reading(samples, self.reading)
        verify_output(samples, self.output)


class TCPReadSilentPeer(BusCase):
    """Hangs the instrument with its connection open. The task warns that the polled
    message missed its periods, invents no samples, and reads again once the
    instrument answers the queries that waited."""

    sim_classes = [SCPISim]
    prefix = "tcp_read_silent"
    reading: Bound

    def create_entries(self) -> list[library.Entry]:
        return [reading_entry()]

    def create_task(self, device: sy.Device, library_key: library.Key) -> tcp.ReadTask:
        reading_msg, self.reading = self.bind_read(
            "reading",
            ["count", "voltage", "current"],
            {"count": sy.DataType.UINT32},
        )
        return tcp.ReadTask(
            name="TCP Read Silent Peer",
            device=device.key,
            library=library_key,
            messages=[reading_msg],
            rate=10 * sy.Rate.HZ,
            timeout=POLL_TIMEOUT,
            framing=bus.DelimiterFraming(delimiter="\n"),
        )

    def run(self) -> None:
        sim = self.simulator(SCPISim)
        with self.statuses() as statuses:
            with self.collect(self.bounds) as samples, self.task.run():
                samples.wait(count=5)
                self.assert_no_problems(statuses)
                sim.hang()
                message = self.wait_for_status(
                    statuses, "warning", containing="reading missed 3 periods"
                )
                self.log(f"Warning while hung: {message}")
                assert_no_new_samples(samples, self.bounds)
                sim.unhang()
                samples.wait_for_more(count=5)
                self.wait_for_status(statuses, "success")
        assert_indexed("reading", self.reading, samples)
        verify_reading(samples, self.reading)
