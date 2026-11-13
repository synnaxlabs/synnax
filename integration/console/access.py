#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Console RBAC UI automation helpers.

These helpers only cover functionality that EXISTS in the Console UI:
- User registration (with role selection)
- The Users page of Settings: change a user's role, rename, and delete

NOT implemented in Console UI (no helpers for these):
- Create, rename, or delete a role
- Create/edit/view policies (hidden from UI)
"""

import re

from playwright.sync_api import Locator, TimeoutError as PlaywrightTimeoutError, expect

import synnax as sy
from console.base import ResourceClient
from console.layout import LayoutClient


class AccessClient(ResourceClient):
    """Console RBAC client for existing role/user UI functionality."""

    USER_ITEM_SELECTOR = ".console-user-members__item"
    USER_SEARCH_PLACEHOLDER = "Search users"

    def __init__(self, layout: LayoutClient):
        super().__init__(layout)
        # Name of the test's project, set by ConsoleCase. Logout clears the
        # active project, so login() re-selects it to return to the app.
        self.bootstrap_project: str | None = None

    # -------------------------------------------------------------------------
    # Login/Logout
    # -------------------------------------------------------------------------

    def logout(self) -> None:
        """Log out the current user via command palette.

        After logout, the login screen will be displayed.
        """
        self.layout.press_key("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.layout.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Log out", timeout=2000)
        sy.sleep(0.2)

        logout_option = self.layout.page.get_by_text("Log out", exact=True).first
        logout_option.click(timeout=2000)
        sy.sleep(0.5)

        self.layout.page.get_by_label("Username", exact=True).wait_for(
            state="visible", timeout=5000
        )

    def logout_via_badge(self) -> None:
        """Log out via the Core badge dropdown.

        Clicks the Core badge, then clicks "Log out" in the dropdown.
        After logout, the login screen will be displayed.
        """
        user_badge = self.layout.page.get_by_role(
            "button", name="Core menu", exact=True
        )
        user_badge.click()
        sy.sleep(0.3)

        logout_btn = self.layout.page.get_by_role("button", name="Log out")
        logout_btn.click()
        sy.sleep(0.5)

        self.layout.page.get_by_label("Username", exact=True).wait_for(
            state="visible", timeout=5000
        )

    def login(self, *, username: str, password: str) -> None:
        """Log in as a user.

        Assumes the login screen is currently displayed.

        :param username: The username to log in with.
        :param password: The password to log in with.
        :raises RuntimeError: If login fails with an error message.
        """
        username_input = self.layout.page.get_by_label("Username", exact=True).first
        username_input.wait_for(state="visible", timeout=5000)
        username_input.fill(username)

        password_input = self.layout.page.get_by_label("Password", exact=True).first
        password_input.fill(password)

        # The button's trailing arrow icon pollutes its accessible name, so exact
        # name matching fails; substring matching still isolates it.
        self.layout.page.get_by_role("button", name="Log in").click()

        for _ in range(20):
            sy.sleep(0.5)

            error_note = self.layout.page.locator(".console-login__error")
            if error_note.count() > 0 and error_note.is_visible():
                error_text = error_note.inner_text().strip()
                raise RuntimeError(f"Login failed: {error_text}")

            login_form = self.layout.page.get_by_label("Username", exact=True)
            if login_form.count() == 0 or not login_form.is_visible():
                self._enter_test_project()
                return

        raise RuntimeError("Login timed out")

    def _enter_test_project(self) -> None:
        """Select the test's project after login if the Splash screen is shown.

        An active project is required to use the app, and logout clears the
        active one, so a fresh login lands on the project Splash screen. This
        selects the test's project to return to the app. A user without
        project-retrieve permission cannot see the project and stays on the
        Splash screen.
        """
        if self.bootstrap_project is None:
            return
        try:
            self.layout.page.wait_for_selector(
                ".console-project-splash, .console-palette button",
                state="visible",
                timeout=10000,
            )
        except PlaywrightTimeoutError:
            return
        if not self.layout.page.locator(".console-project-splash").is_visible():
            return
        item = (
            self.layout.page.locator(".console-project-splash__list")
            .get_by_text(self.bootstrap_project, exact=True)
            .first
        )
        try:
            item.wait_for(state="visible", timeout=5000)
        except PlaywrightTimeoutError:
            return
        item.click(timeout=5000)
        self.layout.page.wait_for_selector(
            ".console-palette button", state="visible", timeout=10000
        )

    def get_current_user(self) -> str | None:
        """Get the username of the currently logged in user.

        :returns: The username, or None if not logged in.
        """
        user_badge = self.layout.page.get_by_role(
            "button", name="Core menu", exact=True
        )
        if user_badge.count() > 0 and user_badge.is_visible():
            return user_badge.inner_text().strip()
        return None

    def get_current_role(self) -> str | None:
        """Get the role name shown in the Core badge dialog.

        Opens the badge dialog, reads the role, and closes it again.

        :returns: The role name, or None if the badge shows none.
        """
        badge = self.layout.page.locator(".lyra-dialog__trigger:has(.lyra-icon--user)")
        if badge.count() == 0 or not badge.is_visible():
            return None
        badge.click()
        role = self.layout.page.locator(".console-user-info__role")
        try:
            role.wait_for(state="visible", timeout=5000)
            return role.inner_text().strip()
        except PlaywrightTimeoutError:
            return None
        finally:
            self.layout.press_escape()

    # -------------------------------------------------------------------------
    # Permission probes
    # -------------------------------------------------------------------------

    def users_page_visible(self) -> bool:
        """Report whether Settings shows the Users page to the current user.

        The page is gated on user update, so only an Owner sees it. Leaves
        Settings closed.
        """
        if not self.layout.command_exists("Open settings"):
            return False
        nav = self.layout.open_settings()
        users = nav.get_by_role("menuitem", name="Users", exact=True)
        try:
            users.wait_for(state="visible", timeout=2000)
            return True
        except PlaywrightTimeoutError:
            return False
        finally:
            self.layout.close_tab("Settings")

    # -------------------------------------------------------------------------
    # User Registration (with role selection)
    # -------------------------------------------------------------------------

    def register_user(
        self,
        *,
        username: str,
        password: str,
        role_name: str,
        name: str | None = None,
    ) -> None:
        """Register a new user via Console UI command palette.

        :param username: The username for the new user.
        :param password: The password for the new user.
        :param role_name: Role to assign to the user (required).
        :param name: Name of the user, which the users panel shows. Defaults to the
            username.
        :raises AssertionError: If registration surfaces an error notification.
        """
        self.notifications.close_all()

        self.layout.command_palette("Register user")

        # Settings can sit behind the dialog with its own "Password" and "Role", so
        # every field lookup stays inside the dialog.
        modal = self.layout.page.locator(self.layout.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        fields = (
            ("Name", name if name is not None else username),
            ("Username", username),
            ("Password", password),
        )
        for label, value in fields:
            modal.locator(f"text={label}").locator("..").locator(
                "input, textarea"
            ).first.fill(value)

        self._role_picker(modal).click()
        self.layout.select_from_dropdown(role_name, placeholder="Search")

        self.layout.page.get_by_role("button", name="Register", exact=True).click()

        sy.sleep(0.5)

        if self.layout.check_for_errors("user"):
            raise AssertionError(f"Failed to register user {username!r}")

    # -------------------------------------------------------------------------
    # Users page of Settings
    # -------------------------------------------------------------------------

    def assign_role_to_user(self, *, username: str, role_name: str) -> None:
        """Change a user's role with the role picker on their line.

        :param username: The name of the user.
        :param role_name: The name of the role to assign.
        """
        self._role_picker(self._get_user_item(username)).click()
        self.layout.select_from_dropdown(role_name, placeholder="Search")

    def wait_for_user_role(
        self, username: str, *, role_name: str, timeout: int = 5000
    ) -> None:
        """Wait until the Users page shows a user with a role.

        :param username: The name of the user.
        :param role_name: The role the user must hold.
        :param timeout: Maximum time to wait, in milliseconds.
        """
        picker = self._role_picker(self._get_user_item(username))
        expect(picker).to_have_text(role_name, timeout=timeout)

    def rename_user(self, *, name: str, new_name: str) -> None:
        """Rename a user via the "Rename" context menu action.

        :param name: The current name of the user.
        :param new_name: The new name of the user.
        """
        self.layout.rename_in_place(self._get_user_item(name), new_name)

    def delete_user(self, username: str) -> None:
        """Delete a user via context menu with confirmation.

        :param username: The name of the user to delete.
        """
        self.layout.delete_with_confirmation(self._get_user_item(username))
        expect(self._user_items(username)).to_have_count(0, timeout=5000)

    def find_user_item(self, name: str) -> Locator | None:
        """Search the Users page for a user.

        :param name: The name of the user.
        :returns: The user's line, or None if the search does not list them.
        """
        self._show_users_page()
        self.layout.page.get_by_placeholder(
            self.USER_SEARCH_PLACEHOLDER, exact=True
        ).fill(name)
        item = self._user_items(name).first
        try:
            item.wait_for(state="visible", timeout=5000)
        except PlaywrightTimeoutError:
            return None
        return item

    # -------------------------------------------------------------------------
    # Helper methods
    # -------------------------------------------------------------------------

    def _show_users_page(self) -> None:
        """Show the Users page of Settings."""
        nav = self.layout.open_settings()
        nav.get_by_role("menuitem", name="Users", exact=True).click()
        self.layout.page.get_by_placeholder(
            self.USER_SEARCH_PLACEHOLDER, exact=True
        ).wait_for(state="visible", timeout=5000)

    def _user_items(self, name: str) -> Locator:
        """The lines on the Users page whose name is ``name``."""
        page = self.layout.page
        return page.locator(self.USER_ITEM_SELECTOR).filter(
            has=page.get_by_text(name, exact=True)
        )

    def _role_picker(self, scope: Locator) -> Locator:
        """The role picker in a dialog or on a user's line."""
        return scope.get_by_role("button", name=re.compile(r"^Role\b"))

    def _get_user_item(self, name: str) -> Locator:
        """Like find_user_item, but raises if the search does not list the user."""
        item = self.find_user_item(name)
        if item is None:
            raise ValueError(f"User '{name}' not found on the Users page")
        return item
