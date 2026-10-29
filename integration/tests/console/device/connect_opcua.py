#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
import ssl

from examples.opcua import OPCUATLSDriverCertificateSim, read_server_certificate

from tests.console.task.opcua_case import OPCUACase
from tests.driver.opcua_task import pinned_certificate
from x import random_name

NETWORK = "OPC UA server"
TIMEOUT = 15000


class ConnectOPCUA(OPCUACase):
    """Connect a secured OPC UA server with the Driver certificate. The Driver rejects
    the server certificate until the user trusts the one that the server sent."""

    sim_classes = [OPCUATLSDriverCertificateSim]

    def run(self) -> None:
        self.log("Testing: Trust the server certificate")
        devices = self.console.devices
        devices.open_connect()
        devices.pick_integration(NETWORK)
        modal = devices.modal
        name = f"Secured server {random_name()}"
        modal.get_by_label("Name", exact=True).fill(name)
        modal.get_by_label(re.compile(r"^Endpoint\b")).fill(
            OPCUATLSDriverCertificateSim.endpoint
        )
        modal.get_by_role("button", name="Sign and encrypt", exact=True).click(
            timeout=5000
        )
        # The Driver answers the certificate request before the dialog shows an expiry.
        modal.get_by_text(re.compile(r"^Expires ")).wait_for(timeout=TIMEOUT)
        # Toasts stack over the modal footer and swallow the click.
        self.console.layout.notifications.close_all()
        modal.get_by_role("button", name="Connect", exact=True).click(timeout=5000)
        modal.get_by_role("button", name="Trust", exact=True).click(timeout=TIMEOUT)
        modal.wait_for(state="hidden", timeout=TIMEOUT)

        dev = self.client.devices.retrieve(name=name)
        self.track_test_devices([dev])
        expected = ssl.PEM_cert_to_DER_cert(read_server_certificate())
        assert pinned_certificate(dev) == expected, (
            f"'{name}' should trust the simulator's certificate"
        )
