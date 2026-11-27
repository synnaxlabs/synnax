#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The login form of a Console that a Core serves."""

from console.case import ConsoleCase


class LoginForms(ConsoleCase):
    """Log in to the serving Core: a bad password shows why, a good one opens the
    Console. The Cores list and the connect modal have Console specs."""

    def setup(self) -> None:
        self._launch_browser()
        self._goto_console()

    def run(self) -> None:
        username = self.page.get_by_label("Username", exact=True).first
        password = self.page.get_by_label("Password", exact=True).first
        log_in = self.page.get_by_role("button", name="Log in")
        self.console.layout.wait_for_visible(username)
        username.fill(self.synnax_connection.username)

        self.log("Submitting bad credentials")
        password.fill("wrong-password")
        log_in.click(timeout=5000)
        self.console.layout.wait_for_visible(
            self.page.get_by_text("Incorrect username or password.")
        )

        self.log("Submitting good credentials")
        password.fill(self.synnax_connection.password)
        log_in.click(timeout=5000)
        self.page.locator(".console-project-splash").wait_for(
            state="visible", timeout=15000
        )
