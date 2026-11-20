#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""An Arc automation on the Driver that runs the EtherCAT bus of the HITL station."""

from typing import Unpack

import synnax as sy
from framework.test_case import Config
from tests.arc.arc import ArcCase
from tests.driver.task import delete_channels
from tests.hitl.ethercat import (
    DO,
    START_TIMEOUT,
    STGS,
    create_read_task,
    create_write_task,
    prepare_bus,
)

BLINK_SOURCE = """
func blink() {count u32} u8 {
    return u8((count / 50) % 2)
}
hitl_ethercat_bufcounter_2 -> blink() -> hitl_ethercat_do_cmd_1
"""


class EtherCATArc(ArcCase):
    """Blinks LED 1 of a 32xDO from the cycle counter of a 6xSTG."""

    arc_source = BLINK_SOURCE
    arc_name_prefix = "EtherCATArc"
    start_cmd_channel = "hitl_ethercat_arc_start_cmd"
    subscribe_channels = ["hitl_ethercat_do_state_1"]
    tasks: list[sy.ethercat.ReadTask | sy.ethercat.WriteTask]

    def __init__(self, **kwargs: Unpack[Config]) -> None:
        super().__init__(**kwargs)
        self.tasks = []

    def setup(self) -> None:
        slaves = prepare_bus(self.client)
        self.rack = self.client.racks.retrieve(slaves[DO].rack)
        write = create_write_task(
            self.client,
            slaves[DO],
            name="HITL EtherCAT Arc Write",
            state_rate=50 * sy.Rate.HZ,
        )
        read = create_read_task(
            self.client,
            [slaves[STGS[0]]],
            name="HITL EtherCAT Arc Read",
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
        )
        for tsk in (write, read):
            self.client.tasks.configure(tsk)
            self.tasks.append(tsk)
            tsk.start(timeout=START_TIMEOUT)
        super().setup()

    def _retrieve_rack(self) -> None:
        self.log(f"Selecting rack: {self.rack.name} (key: {self.rack.key})")

    def verify_sequence_execution(self) -> None:
        for state in (1, 0):
            self.wait_for_eq("hitl_ethercat_do_state_1", state)

    def teardown(self) -> None:
        channels: list[int] = []
        try:
            for tsk in self.tasks:
                if isinstance(tsk, sy.ethercat.ReadTask):
                    channels += [ch.channel for ch in tsk.config.channels]
                else:
                    for w in tsk.config.channels:
                        channels += [w.cmd_channel, w.state_channel]
                with self._try_to(f"delete task {tsk.name}"):
                    self.client.tasks.delete(tsk.key)
        finally:
            super().teardown()
            # The Arc writes a command channel until super().teardown() deletes it.
            with self._try_to("delete task channels"):
                delete_channels(self.client, channels)
