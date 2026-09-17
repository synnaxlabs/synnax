#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The desktop-style login flow: cluster list, connect modal, credentials."""

from playwright.sync_api import Locator

from console.case import ConsoleCase
from x import random_name


class LoginForms(ConsoleCase):
    """Exercise the cluster selection step of Login and its connect modal.

    The web build skips cluster selection because it detects the serving Core;
    the ``select-core`` query param disables that detection so the full
    desktop-style flow renders.
    """

    def setup(self) -> None:
        self._launch_browser()
        self._goto_console(query="?select-core")

    def _wait(self, locator: Locator) -> None:
        self.console.layout.wait_for_visible(locator)

    def run(self) -> None:
        self.suffix = random_name()
        self.core_name = f"TestCore_{self.suffix}"
        self.test_cluster_list_step()
        self.test_connect_modal_validation()
        self.test_add_core()
        self.test_rename_core()
        self.test_refresh_core()
        self.test_copy_link()
        self.test_remove_core()
        self.test_login_round_trip()

    def test_cluster_list_step(self) -> None:
        """Login starts at the cluster list with the seeded Cores."""
        self.log("Waiting for the cluster list")
        cluster = self.console.cluster
        self._wait(self.page.get_by_text("Cores", exact=True))
        self._wait(cluster.core_item("Local"))
        self._wait(cluster.core_item("Demo"))

    def test_connect_modal_validation(self) -> None:
        """An empty submit is rejected with field-level requirements."""
        self.log("Submitting an empty connect modal")
        cluster = self.console.cluster
        self.page.get_by_text("Add a Core").click(timeout=5000)
        self._wait(cluster.modal)
        cluster.modal.get_by_role("button", name="Connect").click(timeout=5000)
        self._wait(cluster.modal.get_by_text("Name is required"))
        self.page.get_by_label("Close").click(timeout=5000)
        self.console.layout.wait_for_hidden(cluster.modal)

    def test_add_core(self) -> None:
        """Adding a Core lists it."""
        self.log(f"Adding Core {self.core_name}")
        self.console.cluster.add_core(
            name=self.core_name,
            host=self.synnax_connection.server_address,
            port=self.synnax_connection.port,
        )

    def test_rename_core(self) -> None:
        """Renaming through the context menu updates the list."""
        new_name = f"Renamed_{self.suffix}"
        self.log(f"Renaming {self.core_name} to {new_name}")
        self.console.cluster.rename_core(old_name=self.core_name, new_name=new_name)
        self.core_name = new_name

    def test_refresh_core(self) -> None:
        """Refresh connection reports success against the live Core."""
        self.log(f"Refreshing connection for {self.core_name}")
        self.console.cluster.refresh_core(self.core_name)
        assert self.console.notifications.wait_for(f"Connected to {self.core_name}"), (
            "refresh should raise a success notification"
        )
        self.console.notifications.close_all()

    def test_copy_link(self) -> None:
        """A link names a cluster, which only a connection can report."""
        self.log(f"Copying a link to the unconnected {self.core_name}")
        link = self.console.cluster.copy_core_link(self.core_name)
        assert link == "", f"expected no link, got {link!r}"
        assert self.console.notifications.wait_for(
            f"Failed to copy link to {self.core_name}"
        ), "copying without a connection should raise an error notification"
        self.console.notifications.close_all()

    def test_remove_core(self) -> None:
        """Removing the added Core takes it out of the list."""
        self.log(f"Removing {self.core_name}")
        self.console.cluster.remove_core(self.core_name)

    def test_login_round_trip(self) -> None:
        """Select a Core, navigate back, reject bad credentials, then log in."""
        cluster = self.console.cluster
        self.log("Selecting Local and checking the credentials step")
        cluster.select_core("Local")
        username = self.page.get_by_label("Username", exact=True).first
        self._wait(username)
        assert username.input_value() == self.synnax_connection.username, (
            "the saved username should prefill"
        )

        back = self.page.locator(".console-login__back")
        self._wait(back)
        back.click(timeout=5000)
        self._wait(cluster.core_item("Local"))

        cluster.select_core("Local")
        self.log("Submitting bad credentials")
        password = self.page.get_by_label("Password", exact=True).first
        password.fill("wrong-password")
        self.page.get_by_role("button", name="Log in").click(timeout=5000)
        self._wait(self.page.locator(".console-login__status .pluto--status-error"))

        self.log("Submitting good credentials")
        password.fill(self.synnax_connection.password)
        self.page.get_by_role("button", name="Log in").click(timeout=5000)
        self.page.locator(".console-project-splash").wait_for(
            state="visible", timeout=15000
        )
