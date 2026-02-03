#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from framework.test_case import SynnaxConnection, TestCase


class SimDaqTestCase(TestCase):
    """
    TestCase with automatic SimDAQ lifecycle management.

    Subclasses should define sim_daq_class as a class attribute to specify
    which simulator to use. The simulator will be automatically started
    during __init__ and stopped during teardown.

    Example:
        class MyTest(SimDaqTestCase):
            sim_daq_class = PressSimDAQ

            def run(self):
                # simulator is already running via self.sim_daq
                ...

    For console tests, use multiple inheritance:

        class MyConsoleTest(SimDaqTestCase, ConsoleCase):
            sim_daq_class = PressSimDAQ
    """

    sim_daq_class: type
    sim_daq: Any

    def __init__(
        self,
        synnax_connection: SynnaxConnection = SynnaxConnection(),
        *,
        name: str,
        **params: Any,
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
        if hasattr(self, "sim_daq") and self.sim_daq is not None:
            self.sim_daq.stop()
        super().teardown()
