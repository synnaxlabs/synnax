#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

from examples.http_sim import HTTPSim

import synnax as sy
from console.case import ConsoleCase
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import create_channel, create_index, delete_channels

STATUS_TIMEOUT = 30 * sy.TimeSpan.SECOND


class HTTPLegacyDevice(SimulatorCase, ConsoleCase):
    """An HTTP device saved by an older Python client still scans, reads, and opens.

    The older client wrote no validate_response or max_concurrent_requests and left
    bad index values in the read map. The sim device beside it has the current shape,
    so the form is checked against each.
    """

    sim_classes = [HTTPSim]

    def setup(self) -> None:
        super().setup()
        suffix = random.randint(1000, 9999)
        rack = self.client.racks.retrieve(name=self.RACK_NAME)
        self.legacy = self.client.devices.create(
            sy.Device(
                key=f"http-legacy-{suffix}",
                rack=rack.key,
                name=f"HTTP Legacy Server {suffix}",
                make="http",
                model="HTTP server",
                location=f"{HTTPSim.host}:{HTTPSim.port}",
                configured=True,
                properties={
                    "secure": False,
                    "verify_ssl": False,
                    "timeout_ms": 5000,
                    "auth": {"type": "none"},
                    "health_check": {"method": "GET", "path": "/health"},
                    # Older clients stored null or a field uuid as the index.
                    "read": {
                        "/api/v1/status": {"index": None, "channels": {}},
                        "/api/v1/metrics": {
                            "index": "1f3c1c1e-3f1a-4d2b-9c8e-0d8a1d2e3f4a",
                            "channels": {},
                        },
                    },
                    "write": {},
                    "version": 1,
                },
            )
        )
        self.track_test_devices([self.legacy])
        self.channel_keys: list[int] = []
        self.tsk: sy.http.ReadTask | None = None

    def teardown(self) -> None:
        with self._try_to("delete the read task"):
            if self.tsk is not None:
                self.client.tasks.delete(self.tsk.key)
        with self._try_to("delete the read task channels"):
            delete_channels(self.client, self.channel_keys)
        super().teardown()

    def run(self) -> None:
        self.test_driver_connects()
        self.test_read_task_reads()
        self.test_edit_connection_opens(self.legacy.name)
        self.test_edit_connection_opens(self.device_name)

    def test_driver_connects(self) -> None:
        """The Driver health checks the legacy device and reports it connected."""
        self.log("Testing: Driver connects the legacy device")
        status_key = str(sy.ontology.ID(type="device", key=self.legacy.key))
        timer = sy.Timer()
        while True:
            statuses = self.client.statuses.retrieve(keys=[status_key])
            if statuses and statuses[0].variant == "success":
                break
            if timer.elapsed() > STATUS_TIMEOUT:
                self.fail(f"Legacy device never connected, last status {statuses}")
            sy.sleep(1)

    def test_read_task_reads(self) -> None:
        """A read task on the legacy device delivers data with the default cap."""
        self.log("Testing: Read task on the legacy device")
        idx = create_index(self.client, "http_legacy_index")
        key = create_channel(
            self.client,
            name="http_legacy_temperature",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        self.channel_keys = [idx.key, key]
        self.tsk = sy.http.ReadTask(
            name="HTTP Legacy Read",
            device=self.legacy.key,
            rate=10,
            endpoints=[
                sy.http.ReadEndpoint(
                    path="/api/v1/data",
                    method="GET",
                    fields=[
                        sy.http.ReadField(
                            pointer="/temperature", channel=key, data_type="float64"
                        )
                    ],
                )
            ],
        )
        self.client.tasks.configure(self.tsk)
        with self.tsk.run():
            with self.client.open_streamer([key]) as streamer:
                frame = streamer.read(timeout=10)
        if frame is None or key not in frame:
            self.fail("Read task delivered no data")

    def test_edit_connection_opens(self, name: str) -> None:
        """The connection form opens on the device with both fields populated."""
        self.log(f"Testing: Edit connection opens for {name}")
        layout = self.console.layout
        item = self.console.devices.get(name)
        self.console.devices.ctx_menu.action(item, "Edit connection")
        modal = self.page.locator(layout.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        modal.get_by_placeholder("/health").wait_for(state="visible", timeout=5000)
        cap = layout.get_input_field("Max concurrent requests")
        assert cap == "6", f"Cap should default to 6, got {cap}"
        assert not layout.get_toggle("Validate response body"), (
            "Validate response should be off"
        )
        layout.close_modal(layout.MODAL_SELECTOR)
