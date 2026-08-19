#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Cold connect against a dead Core and recovery through the trouble card."""

import socket

from console.case import ConsoleCase
from console.schematic.schematic import Schematic
from x import random_name


def _dead_port() -> int:
    """A port with nothing listening: bound once to reserve it, then released."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


class ClusterSwitch(ConsoleCase):
    """Repoint the cluster at a dead address, assert the cold takeover, and
    recover through the trouble card's edit action."""

    def run(self) -> None:
        self.suffix = random_name()
        self.page_name = f"switch_page_{self.suffix}"
        self.test_switch_to_dead_core()
        self.test_recover_from_trouble()

    def test_switch_to_dead_core(self) -> None:
        """Pointing at an unreachable Core replaces the workspace with the
        connection takeover."""
        console = self.console
        console.pages.create(Schematic, self.page_name)
        self._cleanup_pages.append(self.page_name)

        dead = _dead_port()
        self.log(f"Switching the cluster to dead port {dead}")
        console.cluster.edit_connection(host="127.0.0.1", port=dead)
        console.cluster.wait_for_trouble()
        console.layout.wait_for_hidden(console.layout.get_tab(self.page_name))
        target = console.cluster.trouble.locator(".console-connection-target")
        assert str(dead) in target.inner_text(), (
            f"trouble card should show the dead address, got {target.inner_text()!r}"
        )

    def test_recover_from_trouble(self) -> None:
        """Editing the connection back to the live Core restores the workspace
        with its tabs intact."""
        console = self.console
        host = self.synnax_connection.server_address
        port = self.synnax_connection.port
        self.log(f"Recovering to {host}:{port} from the trouble card")
        console.cluster.edit_connection_from_trouble(host=host, port=port)
        console.layout.wait_for_hidden(console.cluster.trouble)
        console.layout.wait_for_tab(self.page_name)
        console.cluster.open_badge()
        console.cluster.wait_for_status("Connected", timeout=20000)
        console.cluster.close_badge()
