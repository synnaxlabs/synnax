#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Cases of the bus integrations, which ship dark."""

from typing import Unpack

import synnax as sy
from framework import drivers
from framework.test_case import Config
from tests.driver.simulator_case import SimulatorCase

INTEGRATIONS = ["serial"]
"""The bus integrations, which the Driver of a run leaves off."""
DRIVER_UP_TIMEOUT = 30 * sy.TimeSpan.SECOND


class BusCase(SimulatorCase):
    """Base class for cases of the bus integrations. The Driver of the run leaves them
    off, so each case runs a standalone Driver of its own that runs only them. The
    devices of the simulators sit on the rack of that Driver."""

    def __init__(self, **kwargs: Unpack[Config]) -> None:
        super().__init__(**kwargs)
        self.driver: drivers.Process | None = None
        self._rack: sy.Rack | None = None

    def setup(self) -> None:
        """Start the Driver and wait until it runs, then delegate to next in MRO."""
        binary = self.options.driver_binary
        if binary is None:
            self.skip("needs --driver-binary")
        self.driver = drivers.Process(
            binary=binary,
            connection=self.synnax_connection,
            log_path=self.results_dir / "driver.log",
            integrations=INTEGRATIONS,
        )
        self._rack = self._wait_for_rack(self.driver)
        super().setup()

    def rack(self) -> sy.Rack:
        assert self._rack is not None, "setup() must run before rack()"
        return self._rack

    def _wait_for_rack(self, driver: drivers.Process) -> sy.Rack:
        """Return the rack of driver once the Driver reports itself running."""
        api_key = sy.poll(driver.api_key, timeout=DRIVER_UP_TIMEOUT, interval=0.2)
        if api_key is None:
            self.fail(f"the Driver did not enroll in {DRIVER_UP_TIMEOUT}")
        key = self._rack_key(api_key)
        self.track_test_racks([self.client.racks.retrieve(key)])

        def running() -> sy.Rack | None:
            rack = self.client.racks.retrieve(key, include_status=True)
            status = rack.status
            if status is None or status.details is None or not status.details.running:
                return None
            return rack

        rack = sy.poll(running, timeout=DRIVER_UP_TIMEOUT, interval=0.2)
        if rack is None:
            self.fail(
                f"the Driver did not report itself running in {DRIVER_UP_TIMEOUT}"
            )
        return rack

    def _rack_key(self, api_key: str) -> int:
        """Return the key of the rack that api_key logs in as."""
        c = self.synnax_connection
        client = sy.Synnax(
            host=c.server_address,
            port=c.port,
            secure=c.secure,
            ca_cert=c.ca_cert,
            api_key=api_key,
        )
        try:
            subject = client.auth.subject
        finally:
            client.close()
        if subject is None or subject.type != "rack" or subject.key is None:
            self.fail(f"the API key of the Driver logs in as {subject}, not a rack")
        return int(subject.key)

    def teardown(self) -> None:
        """Stop the Driver, then run the remaining teardown, which deletes its rack."""
        try:
            if self.driver is not None:
                with self._try_to("stop the Driver"):
                    self.driver.stop()
        finally:
            super().teardown()
