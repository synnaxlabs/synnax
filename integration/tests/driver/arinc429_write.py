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
from synnax import arinc429, library
from tests.driver.bus_task import assert_equal
from tests.driver.card_task import ARINC429Card, CardCase, wait_for_value

LABEL = 0o310
# Commands for a signed 16-bit field at a quarter unit per count, which reaches the
# sign bit of the word's data.
COMMANDS = [1.5, -2.0, 100.25, -8191.75]


class ARINC429Write(CardCase):
    """Sends a label once for each command and checks that a peer read task on the
    same channel decodes each command once, in order."""

    device_classes = [ARINC429Card]
    prefix = "arinc429_write"
    command_channel: int

    def create_entries(self) -> list[library.Entry]:
        return [
            library.MessageEntry(
                name="heading",
                identifier=library.Arinc429Identifier(
                    label=LABEL, sdi=3, sdi_matched=True
                ),
                fields=[
                    library.BinaryField(
                        name="value",
                        start_bit=12,
                        bit_length=16,
                        signed=True,
                        scale=0.25,
                    )
                ],
            )
        ]

    def create_task(
        self, device: sy.Device, library_key: library.Key
    ) -> arinc429.WriteTask:
        message, channels = self.bind_write("heading", ["value"])
        self.command_channel = channels["value"]
        return arinc429.WriteTask(
            name="ARINC 429 Write",
            device=device.key,
            library=library_key,
            messages=[message],
        )

    def run(self) -> None:
        message, bound = self.bind_read("heading", ["value"])
        reader = self.configure_peer(
            arinc429.ReadTask(
                name="ARINC 429 Write Peer",
                device=self.device(ARINC429Card).key,
                library=self.library_key,
                messages=[message],
            )
        )
        with self.statuses() as statuses:
            with self.collect([bound]) as samples, self.run_all(reader):
                for value in COMMANDS:
                    self.command_until(
                        {self.command_channel: value},
                        partial(
                            wait_for_value,
                            samples,
                            bound.fields["value"],
                            value,
                            2 * sy.TimeSpan.SECOND,
                        ),
                    )
                samples.drain()
            self.assert_no_problems(statuses)
        assert_equal("heading.value", samples[bound.fields["value"]], COMMANDS)
