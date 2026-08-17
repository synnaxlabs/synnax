#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that Engineer role has full access except user management."""

from console.case import ConsoleCase
from x import random_name

PASSWORD = "testpassword123"
FIRST_NAME = "Engineer"
SCHEMATIC_NAME = "engineer_perm_schematic"


class RoleEngineerPermissions(ConsoleCase):
    """Engineer holds every action except user, role, and policy writes."""

    def run(self) -> None:
        self.log_in_as_engineer()
        self.badge_names_the_role()
        self.user_management_is_hidden()
        self.creation_commands_are_offered()
        self.mosaic_is_interactive()

    def log_in_as_engineer(self) -> None:
        self._username = f"engineer_{random_name()}"
        assert self.console.access.register_user(
            username=self._username,
            password=PASSWORD,
            first_name=FIRST_NAME,
            last_name="Test",
            role_name="Engineer",
        ), f"failed to register user {self._username}"
        self.console.access.logout()
        self.console.access.login(username=self._username, password=PASSWORD)
        self.page.get_by_text(FIRST_NAME, exact=True).wait_for(
            state="visible", timeout=10000
        )

    def badge_names_the_role(self) -> None:
        role = self.console.access.get_current_role()
        assert role == "Engineer", f"badge shows role {role!r}, expected 'Engineer'"

    def user_management_is_hidden(self) -> None:
        """The toolbar needs user update, which an Engineer does not hold."""
        assert not self.console.access.users_toolbar_visible(), (
            "Users toolbar is visible to an Engineer; it is gated on user update"
        )

    def creation_commands_are_offered(self) -> None:
        for command in (
            "Create a project",
            "Create a schematic",
            "Create a line plot",
            "Create a channel",
            "Define a range",
        ):
            assert self.console.access.command_available(command), (
                f"{command!r} is withheld from an Engineer, who can create one"
            )

    def mosaic_is_interactive(self) -> None:
        """An Engineer writes panels, so every structural affordance stays."""
        schematic = self.console.project.create_schematic(SCHEMATIC_NAME)
        self._cleanup_pages.append(schematic.page_name)
        assert self.console.layout.tab_is_closable(SCHEMATIC_NAME), (
            "tab withholds its close button from an Engineer, who can write the panel"
        )
        assert not self.console.layout.mosaic_is_static(), (
            "mosaic withholds structural writes from an Engineer"
        )
