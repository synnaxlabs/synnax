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
from synnax import arinc429, library
from tests.driver.bus_task import Bound, assert_equal, assert_indexed
from tests.driver.card_task import ARINC429Card, CardCase, wait_for_value

ALTITUDE_LABEL = 0o203
STATUS_LABEL = 0o270
PERIOD = 20 * sy.TimeSpan.MILLISECOND
ALTITUDE = -1234.5
OTHER_ALTITUDE = 999.0
MODE = 5


class ARINC429Read(CardCase):
    """Decodes two labels into channels with an index each. A peer write task sends
    them on the same channel every period, along with the altitude label on another
    SDI, which the task must ignore. The status label does not match its SDI, so its
    field may use bits 9 and 10."""

    device_classes = [ARINC429Card]
    prefix = "arinc429_read"
    altitude: Bound
    status: Bound

    def create_entries(self) -> list[library.Entry]:
        def altitude(name: str, sdi: int) -> library.MessageEntry:
            return library.MessageEntry(
                name=name,
                identifier=library.Arinc429Identifier(
                    label=ALTITUDE_LABEL, sdi=sdi, sdi_matched=True
                ),
                period=PERIOD,
                fields=[
                    library.BinaryField(
                        name="value",
                        start_bit=10,
                        bit_length=18,
                        signed=True,
                        scale=0.5,
                    )
                ],
            )

        return [
            altitude("altitude", 1),
            altitude("altitude_sdi2", 2),
            library.MessageEntry(
                name="status",
                identifier=library.Arinc429Identifier(label=STATUS_LABEL),
                period=PERIOD,
                fields=[library.BinaryField(name="mode", start_bit=8, bit_length=3)],
            ),
        ]

    def create_task(
        self, device: sy.Device, library_key: library.Key
    ) -> arinc429.ReadTask:
        altitude, self.altitude = self.bind_read("altitude", ["value"])
        status, self.status = self.bind_read("status", ["mode"])
        return arinc429.ReadTask(
            name="ARINC 429 Read",
            device=device.key,
            library=library_key,
            messages=[altitude, status],
        )

    def run(self) -> None:
        messages, commands = [], {}
        for name, field in [
            ("altitude", "value"),
            ("altitude_sdi2", "value"),
            ("status", "mode"),
        ]:
            message, channels = self.bind_write(name, [field])
            messages.append(message)
            commands[name] = channels[field]
        writer = self.configure_peer(
            arinc429.WriteTask(
                name="ARINC 429 Read Peer",
                device=self.device(ARINC429Card).key,
                library=self.library_key,
                messages=messages,
            )
        )
        values = {
            commands["altitude"]: ALTITUDE,
            commands["altitude_sdi2"]: OTHER_ALTITUDE,
            commands["status"]: MODE,
        }
        with self.statuses() as statuses:
            with self.collect([self.altitude, self.status]) as samples:
                with self.run_all(writer):
                    self.command_until(
                        values,
                        lambda: wait_for_value(
                            samples,
                            self.altitude.fields["value"],
                            ALTITUDE,
                            2 * sy.TimeSpan.SECOND,
                        ),
                    )
                    samples.drain()
                    start = samples.count(self.altitude)
                    samples.wait(count=start + 20)
            self.assert_no_problems(statuses)
        assert_indexed("altitude", self.altitude, samples)
        assert_indexed("status", self.status, samples)
        altitudes = samples[self.altitude.fields["value"]][start:]
        assert_equal("altitude.value", altitudes, [ALTITUDE] * len(altitudes))
        modes = samples[self.status.fields["mode"]]
        if not np.any(modes == MODE):
            raise AssertionError(f"status.mode: expected {MODE}, got {set(modes)}")
