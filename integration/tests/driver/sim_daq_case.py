#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
SimDAQ lifecycle mixin.

Provides automatic SimDAQ (thread-based simulator) start/stop management.
Designed for multiple inheritance with TestCase subclasses.

Usage standalone:

    class SimplePress(SimDaqCase, TestCase):
        sim_daq_class = PressSimDAQ

        def run(self):
            # simulator is already running via self.sim_daq
            ...

Usage with ConsoleCase:

    class MyConsoleTest(SimDaqCase, ConsoleCase):
        sim_daq_class = PressSimDAQ
"""

from examples.simulators.simdaq import SimDAQ

from framework.models import SynnaxConnection
from framework.test_case import TestCase


class SimDaqCase(TestCase):
    """Mixin for SimDAQ lifecycle management.

    Class attributes:
        sim_daq_class: SimDAQ subclass to instantiate
    """

    sim_daq_class: type[SimDAQ] | None = None
    sim_daq: SimDAQ | None

    def __init__(
        self,
        synnax_connection: SynnaxConnection = SynnaxConnection(),
        *,
        name: str,
        **params: object,
    ) -> None:
        super().__init__(synnax_connection, name=name, **params)
        # Not all test cases need a simulator (e.g. edge_cases).
        sim_cls = self.sim_daq_class
        if sim_cls is not None:
            self.sim_daq = sim_cls(self.client)
            self.sim_daq.start()
        else:
            self.sim_daq = None

    def teardown(self) -> None:
        """Stop the simulator, then run the remaining teardown.

        Stopping is done first and unconditionally so a failure in a later
        teardown step cannot leave the simulator's writer open. A leaked writer
        keeps control authority over shared channels and fails every later
        simulator-backed test.
        """
        try:
            if self.sim_daq is not None:
                self.sim_daq.stop()
        finally:
            super().teardown()
