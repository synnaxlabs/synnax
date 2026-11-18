#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A startup parameter that a 6xSTG of the HITL station refuses."""

import synnax as sy
from framework.test_case import TestCase
from tests.driver.task import assert_start_rejected, delete_channels
from tests.hitl.ethercat import SLAVES, STGS, create_read_task, prepare_bus

MISSING = sy.ethercat.StartupParameter(
    index=0x8040, sub_index=0xFF, data_type=sy.DataType.UINT8, value=0
)
"""A sub-index that the settings of input 1 of a 6xSTG do not hold."""

REJECT_TIMEOUT = 15 * sy.TimeSpan.SECOND
"""Well under the minute of start retries, which a retried abort would use up."""


class EtherCATRefusedStartupParameter(TestCase):
    """Checks that a start with a startup parameter at a missing address fails at once
    with the abort reason of the slave."""

    tsk: sy.ethercat.ReadTask | None = None

    def setup(self) -> None:
        self.stg = prepare_bus(self.client)[STGS[0]]

    def run(self) -> None:
        self.tsk = create_read_task(
            self.client,
            [self.stg],
            name="HITL EtherCAT Refused Startup Parameter",
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            startup_parameters=[
                sy.ethercat.SlaveStartupParameters(
                    device=self.stg.key, parameters=[MISSING]
                )
            ],
        )
        text = assert_start_rejected(
            self.client, self.tsk, "a missing address", timeout=REJECT_TIMEOUT
        )
        expected = (
            f"Failed to start task: Module {STGS[0]} ({SLAVES[STGS[0] - 1]}) rejected "
            "startup parameter 0x8040:FF: Subindex does not exist."
        )
        if text != expected:
            self.fail(f"Expected {expected!r}, but got {text!r}")

    def teardown(self) -> None:
        if self.tsk is not None:
            with self._try_to("delete the task channels"):
                delete_channels(
                    self.client, [ch.channel for ch in self.tsk.config.channels]
                )
        super().teardown()
