#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from functools import partial

import synnax as sy
from synnax import library, mil1553
from tests.driver.bus_task import Bound, assert_equal, assert_indexed
from tests.driver.card_task import (
    REMOTE_TERMINAL,
    BusController,
    CardCase,
    RemoteTerminal,
    wait_for_value,
)

PERIOD = 20 * sy.TimeSpan.MILLISECOND
# Rounds of the words the remote terminal answers with: a count and a signed
# temperature, each one big-endian data word.
ROUNDS = [(1234.0, -40.0), (65535.0, 125.0)]


class MIL1553Read(CardCase):
    """Polls a remote terminal as the bus controller every period and decodes the two
    words it answers with. A peer write task on a remote terminal channel of the same
    bus sets those words. The terminal answers busy until it has words, so the task
    may warn at first."""

    sim_classes = [BusController, RemoteTerminal]
    prefix = "mil1553_read"
    status: Bound

    def create_entries(self) -> list[library.Entry]:
        return [
            library.MessageEntry(
                name="status",
                identifier=library.Mil1553Identifier(
                    rt=REMOTE_TERMINAL, subaddress=1, direction="transmit", word_count=2
                ),
                period=PERIOD,
                fields=[
                    library.BinaryField(
                        name="count",
                        start_bit=7,
                        bit_length=16,
                        byte_order="big_endian",
                    ),
                    library.BinaryField(
                        name="temperature",
                        start_bit=23,
                        bit_length=16,
                        byte_order="big_endian",
                        signed=True,
                    ),
                ],
            )
        ]

    def create_task(
        self, device: sy.Device, library_key: library.Key
    ) -> mil1553.ReadTask:
        message, self.status = self.bind_read("status", ["count", "temperature"])
        return mil1553.ReadTask(
            name="MIL-STD-1553 Read",
            device=device.key,
            library=library_key,
            messages=[message],
        )

    def run(self) -> None:
        message, commands = self.bind_write("status", ["count", "temperature"])
        terminal = self.configure_peer(
            mil1553.WriteTask(
                name="MIL-STD-1553 Remote Terminal",
                device=self.device(RemoteTerminal).key,
                library=self.library_key,
                messages=[message],
            )
        )
        count = self.status.fields["count"]
        temperature = self.status.fields["temperature"]
        with self.collect([self.status]) as samples, self.run_all(terminal):
            for c, t in ROUNDS:
                self.command_until(
                    {commands["count"]: c, commands["temperature"]: t},
                    partial(wait_for_value, samples, count, c, 2 * sy.TimeSpan.SECOND),
                )
                samples.drain()
                start = samples.count(self.status)
                samples.wait(count=start + 10)
                counts = samples[count][start:]
                assert_equal("status.count", counts, [c] * len(counts))
                temperatures = samples[temperature][start:]
                assert_equal(
                    "status.temperature", temperatures, [t] * len(temperatures)
                )
        assert_indexed("status", self.status, samples)
