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

from framework.test_case import SynnaxConnection, TestCase


class SimDaqCase(TestCase):
    """Mixin for SimDAQ lifecycle management.

    Class attributes:
        sim_daq_class: SimDAQ subclass to instantiate
    """

    sim_daq_class: type[SimDAQ]
    sim_daq: SimDAQ

    def __init__(
        self,
        synnax_connection: SynnaxConnection = SynnaxConnection(),
        *,
        name: str,
        **params: object,
    ) -> None:
        super().__init__(synnax_connection, name=name, **params)
        if not hasattr(self, "sim_daq_class") or self.sim_daq_class is None:
            raise TypeError(
                f"{self.__class__.__name__} must define 'sim_daq_class' class attribute"
            )
        self.sim_daq = self.sim_daq_class(self.client)
        self.sim_daq.start()

    def teardown(self) -> None:
        """Stop the simulator during teardown."""
        super().teardown()
        if hasattr(self, "sim_daq") and self.sim_daq is not None:
            self.sim_daq.stop()
