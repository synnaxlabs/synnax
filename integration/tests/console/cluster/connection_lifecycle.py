#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Warm Core outage and recovery through a severable TCP proxy."""

import synnax as sy
from console.case import ConsoleCase
from console.schematic.schematic import Schematic
from framework.proxy import SeverableProxy
from x import random_name


class ConnectionLifecycle(ConsoleCase):
    """Sever and restore the link to the Core, asserting the badge state
    machine, workspace survival, and post-reconnect reconciliation."""

    def setup(self) -> None:
        super().setup()
        self.proxy = SeverableProxy(
            self.synnax_connection.server_address, self.synnax_connection.port
        )
        self.recon_channel: str | None = None

    def teardown(self) -> None:
        # Page and project cleanup drive the browser, which talks to the Core
        # through the proxy, so the link must be up before cleanup runs.
        try:
            self.proxy.restore()
        except OSError as e:
            self.log(f"Failed to restore proxy for teardown: {e}")
        try:
            super().teardown()
        finally:
            self.proxy.close()
            if self.recon_channel is not None:
                try:
                    self.client.channels.delete(self.recon_channel)
                except Exception as e:
                    self.log(f"Failed to delete channel {self.recon_channel}: {e}")

    def run(self) -> None:
        self.suffix = random_name()
        self.page_name = f"conn_page_{self.suffix}"
        self.test_connect_through_proxy()
        self.test_outage_degrades_badge()
        self.test_reconnect_reconciles()

    def test_connect_through_proxy(self) -> None:
        """Repoint the active cluster at the proxy through the connect modal."""
        self.log("Connecting through the severable proxy")
        console = self.console
        console.pages.create(Schematic, self.page_name)
        self._cleanup_pages.append(self.page_name)
        # The channels toolbar mounts a subscribed channel query; it stays open
        # across the outage so reconciliation can be asserted without any user
        # action after reconnect.
        console.channels.show_channels()

        console.cluster.edit_connection(host="127.0.0.1", port=self.proxy.port)
        console.cluster.open_badge()
        console.cluster.wait_for_status("Connected", timeout=20000)
        address = console.cluster.address()
        assert str(self.proxy.port) in address, (
            f"badge should show the proxy address, got {address!r}"
        )
        console.cluster.close_badge()

    def test_outage_degrades_badge(self) -> None:
        """A severed link degrades the badge but never unmounts the workspace."""
        self.log("Severing the link to the Core")
        console = self.console
        self.proxy.sever()

        console.cluster.open_badge()
        console.cluster.wait_for_status("Connecting", timeout=10000)
        console.cluster.wait_for_status("Unreachable", timeout=30000)
        console.layout.wait_for_visible(
            console.cluster.badge_dialog.locator(".console-connection-retry")
        )

        assert not console.cluster.trouble.is_visible(), (
            "a warm outage must never show the connection takeover"
        )
        assert console.layout.get_tab(self.page_name).is_visible(), (
            "the workspace should survive a warm outage"
        )

        self.recon_channel = f"recon_{self.suffix}"
        self.log(f"Creating {self.recon_channel} server-side during the outage")
        self.client.channels.create(
            name=self.recon_channel, data_type="float32", virtual=True
        )

    def test_reconnect_reconciles(self) -> None:
        """After restore, the badge recovers and subscribed queries refetch."""
        self.log("Restoring the link to the Core")
        console = self.console
        self.proxy.restore()

        console.cluster.retry_now()
        console.cluster.wait_for_status("Connected", timeout=20000)
        console.cluster.close_badge()

        assert self.recon_channel is not None
        # The channel was created while the link was down; the open toolbar
        # must adopt it purely through the reconnect refetch. The tree is
        # windowed, so sweep it through the helper instead of expecting the
        # item's row in the viewport.
        assert console.channels.wait_for_channels(
            self.recon_channel, timeout=15 * sy.TimeSpan.SECOND
        ), f"{self.recon_channel} should appear via the reconnect refetch"
        assert console.layout.get_tab(self.page_name).is_visible(), (
            "the workspace should survive the full outage cycle"
        )
