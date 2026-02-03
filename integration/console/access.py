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
- Assigning roles to users (context menu → modal)
- Drag-drop user onto role
- Rename/delete roles via context menu (non-internal only)

NOT implemented in Console UI (no helpers for these):
- Create role (no UI exists)
- Create/edit/view policies (hidden from UI)
"""

import synnax as sy
from playwright.sync_api import Locator

from .base import BaseClient
from .layout import LayoutClient


class AccessClient(BaseClient):
    """Console RBAC client for existing role/user UI functionality."""

    SHORTCUT_KEY = "u"
    ROLE_ITEM_PREFIX = "role:"
    USER_ITEM_PREFIX = "user:"

    def __init__(self, layout: LayoutClient):
        super().__init__(layout)

    # -------------------------------------------------------------------------
    # Login/Logout
    # -------------------------------------------------------------------------

    def logout(self) -> None:
        """Log out the current user via command palette.

        After logout, the login screen will be displayed.
        """
        # Open command palette
        self.layout.press_key("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        # Type the logout command
        palette_input = self.layout.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Log Out", timeout=2000)
        sy.sleep(0.2)

        # Click on the Log Out option
        logout_option = self.layout.page.get_by_text("Log Out", exact=True).first
        logout_option.click(timeout=2000)
        sy.sleep(0.5)

        # Wait for login screen to appear
        self.layout.page.wait_for_selector(".pluto-field__username", timeout=5000)

    def logout_via_badge(self) -> None:
        """Log out via the user badge dropdown.

        Clicks on the user badge, then clicks "Log out" in the dropdown.
        After logout, the login screen will be displayed.
        """
        # Click on the user badge (it's a Dialog.Trigger with user icon)
        user_badge = self.layout.page.locator(".pluto-dialog__trigger").filter(
            has=self.layout.page.locator(".pluto-icon--user")
        )
        user_badge.click()
        sy.sleep(0.3)

        # Click "Log out" button in the dropdown
        logout_btn = self.layout.page.get_by_role("button", name="Log out")
        logout_btn.click()
        sy.sleep(0.5)

        # Wait for login screen to appear
        self.layout.page.wait_for_selector(".pluto-field__username", timeout=5000)

    def login(self, *, username: str, password: str) -> None:
        """Log in as a user.

        Assumes the login screen is currently displayed.

        :param username: The username to log in with.
        :param password: The password to log in with.
        :raises RuntimeError: If login fails with an error message.
        """
        # Fill username field (using same selector as ConsoleCase)
        username_input = self.layout.page.locator(".pluto-field__username input").first
        username_input.wait_for(state="visible", timeout=5000)
        username_input.fill(username)

        # Fill password field
        password_input = self.layout.page.locator(".pluto-field__password input").first
        password_input.fill(password)

        # Click Log In button
        self.layout.page.get_by_role("button", name="Log In", exact=True).click()

        # Wait for either success or error
        for _ in range(20):  # 20 * 0.5s = 10s timeout
            sy.sleep(0.5)

            # Check for error message
            error_status = self.layout.page.locator(".pluto-status--error")
            if error_status.count() > 0 and error_status.is_visible():
                error_text = error_status.inner_text().strip()
                raise RuntimeError(f"Login failed: {error_text}")

            # Check if login form disappeared (we're logged in)
            login_form = self.layout.page.locator(".pluto-field__username")
            if login_form.count() == 0 or not login_form.is_visible():
                return  # Success - login form gone

        raise RuntimeError("Login timed out")

    def get_current_user(self) -> str | None:
        """Get the username of the currently logged in user.

        :returns: The username, or None if not logged in.
        """
        # Look for user badge in the UI - it shows the current user
        user_badge = self.layout.page.locator(".console-user-badge")
        if user_badge.count() > 0 and user_badge.is_visible():
            return user_badge.inner_text().strip()
        return None

    # -------------------------------------------------------------------------
    # User Registration (with role selection)
    # -------------------------------------------------------------------------

    def register_user(
        self,
        *,
        username: str,
        password: str,
        first_name: str,
        last_name: str,
        role_name: str,
    ) -> bool:
        """Register a new user via Console UI command palette.

        :param username: The username for the new user.
        :param password: The password for the new user.
        :param first_name: First name of the user.
        :param last_name: Last name of the user.
        :param role_name: Role to assign to the user (required).
        :returns: True if the user was created successfully.
        """
        # Clear any existing notifications to avoid false positives
        self.notifications.close_all()

        # Open command palette and register user
        self.layout.command_palette("Register a User")

        sy.sleep(0.3)

        # Fill first name
        self.layout.fill_input_field("First", first_name)

        # Fill last name
        self.layout.fill_input_field("Last", last_name)

        # Fill username
        self.layout.fill_input_field("Username", username)

        # Fill password
        self.layout.fill_input_field("Password", password)

        # Select role (required)
        self.layout.click_btn("Role")
        self.layout.select_from_dropdown(role_name, placeholder="Search")

        # Click Register button
        self.layout.page.get_by_role("button", name="Register", exact=True).click()

        sy.sleep(0.5)

        # Check for error notifications
        if self._check_for_errors():
            return False

        return True

    # -------------------------------------------------------------------------
    # Role Assignment (context menu → modal)
    # -------------------------------------------------------------------------

    def assign_role_to_user(self, *, username: str, role_name: str) -> bool:
        """Assign a role to a user via the context menu modal.

        This uses the "Assign to role" context menu option on a user,
        which opens a modal with a role dropdown.

        :param username: The username of the user.
        :param role_name: The name of the role to assign.
        :returns: True if successful.
        """
        # Show users panel
        self._show_users_panel()

        # Find the user in the tree
        user_item = self._find_user_item(username)
        if user_item is None:
            raise ValueError(f"User '{username}' not found in users panel")

        # Right-click to open context menu and click "Assign to role"
        self.ctx_menu.open_on(user_item)
        assign_option = self.layout.page.get_by_text("Assign to role", exact=True).first
        if assign_option.count() == 0:
            self.ctx_menu.close()
            raise ValueError("'Assign to role' option not available for this user")
        self.ctx_menu.click_option("Assign to role")

        # Modal should now be open - select role
        if not self.layout.check_for_modal():
            raise RuntimeError("Assign role modal did not open")

        # Click on role dropdown and select
        self.layout.click_btn("Role")
        self.layout.select_from_dropdown(role_name, placeholder="Search")

        # Click Assign button
        self.layout.page.get_by_role("button", name="Assign", exact=True).click()
        sy.sleep(0.3)

        return True

    def drag_user_to_role(self, username: str, role_name: str) -> bool:
        """Assign a role to a user via drag-drop in the ontology tree.

        :param username: The username of the user to drag.
        :param role_name: The name of the role to drop onto.
        :returns: True if successful.
        """
        # Show users panel
        self._show_users_panel()

        # Find user element
        user_item = self._find_user_item(username)
        if user_item is None:
            raise ValueError(f"User '{username}' not found")

        # Find role element - roles are in the ontology tree
        role_item = self._find_role_item(role_name)
        if role_item is None:
            raise ValueError(f"Role '{role_name}' not found")

        # Perform drag and drop
        user_item.drag_to(role_item)
        sy.sleep(0.3)

        return True

    # -------------------------------------------------------------------------
    # Role Rename/Delete (context menu)
    # -------------------------------------------------------------------------

    def rename_role(self, *, old_name: str, new_name: str) -> bool:
        """Rename a role via context menu.

        Note: Internal/system roles cannot be renamed.

        :param old_name: The current name of the role.
        :param new_name: The new name for the role.
        :returns: True if successful.
        """
        role_item = self._find_role_item(old_name)
        if role_item is None:
            raise ValueError(f"Role '{old_name}' not found")

        # Right-click to open context menu
        self.ctx_menu.open_on(role_item)

        # Check if Rename option exists and is not disabled
        rename_option = self.layout.page.get_by_text("Rename", exact=True).first
        if rename_option.count() == 0:
            self.ctx_menu.close()
            raise ValueError("Rename option not available (role may be internal)")
        rename_class = rename_option.get_attribute("class") or ""
        if "disabled" in rename_class.lower():
            self.ctx_menu.close()
            raise ValueError("Rename option is disabled (role may be internal)")

        self.ctx_menu.click_option("Rename")

        # Find the editable text element and fill new name
        role_name_element = role_item.locator("p.pluto-text--editable")
        role_name_element.click()
        role_name_element.fill(new_name)
        self.layout.press_enter()
        sy.sleep(0.2)

        return True

    def delete_role(self, name: str) -> bool:
        """Delete a role via context menu.

        Note: Internal/system roles cannot be deleted.

        :param name: The name of the role to delete.
        :returns: True if successful.
        """
        role_item = self._find_role_item(name)
        if role_item is None:
            raise ValueError(f"Role '{name}' not found")

        # Right-click to open context menu
        self.ctx_menu.open_on(role_item)

        # Check if Delete option exists and is not disabled
        delete_option = self.layout.page.get_by_text("Delete", exact=True).first
        if delete_option.count() == 0:
            self.ctx_menu.close()
            raise ValueError("Delete option not available (role may be internal)")
        delete_class = delete_option.get_attribute("class") or ""
        if "disabled" in delete_class.lower():
            self.ctx_menu.close()
            raise ValueError("Delete option is disabled (role may be internal)")

        self.ctx_menu.click_option("Delete")

        # Confirm deletion in modal if present
        if self.layout.check_for_modal():
            self.layout.page.get_by_role(
                "button", name="Delete", exact=True
            ).first.click()
            sy.sleep(0.3)

        # Check for error notifications
        if self._check_for_errors():
            return False

        return True

    def is_role_modifiable(self, name: str) -> bool:
        """Check if a role can be renamed/deleted (i.e., is not internal).

        :param name: The name of the role to check.
        :returns: True if the role can be modified, False if internal.
        """
        role_item = self._find_role_item(name)
        if role_item is None:
            raise ValueError(f"Role '{name}' not found")

        # Right-click to open context menu
        self.ctx_menu.open_on(role_item)

        # Check if Rename and Delete are available and not disabled
        rename_option = self.layout.page.get_by_text("Rename", exact=True).first
        delete_option = self.layout.page.get_by_text("Delete", exact=True).first

        rename_available = rename_option.count() > 0
        delete_available = delete_option.count() > 0

        if rename_available:
            rename_class = rename_option.get_attribute("class") or ""
            rename_available = "disabled" not in rename_class.lower()

        if delete_available:
            delete_class = delete_option.get_attribute("class") or ""
            delete_available = "disabled" not in delete_class.lower()

        # Close context menu
        self.ctx_menu.close()

        return rename_available and delete_available

    # -------------------------------------------------------------------------
    # Helper methods
    # -------------------------------------------------------------------------

    def _show_users_panel(self) -> None:
        """Show the users panel in the navigation drawer."""
        self._show_toolbar(self.SHORTCUT_KEY, self.ROLE_ITEM_PREFIX)

    def _find_user_item(self, username: str) -> Locator | None:
        """Find a user item in the users panel by username."""
        self._show_users_panel()
        return self.tree.find_by_name(self.USER_ITEM_PREFIX, username)

    def _find_role_item(self, role_name: str) -> Locator | None:
        """Find a role item in the ontology tree by name."""
        self._show_users_panel()
        return self.tree.find_by_name(self.ROLE_ITEM_PREFIX, role_name)

    def list_visible_roles(self) -> list[str]:
        """List all roles visible in the ontology tree.

        :returns: List of role names.
        """
        self._show_users_panel()
        return self.tree.list_names(self.ROLE_ITEM_PREFIX)

    def expand_role(self, role_name: str) -> None:
        """Expand a role node to show its child users.

        :param role_name: The name of the role to expand.
        """
        self._show_users_panel()
        role_item = self._find_role_item(role_name)
        if role_item is None:
            raise ValueError(f"Role '{role_name}' not found")
        self.tree.expand(role_item)

    def list_users_under_role(self, role_name: str) -> list[str]:
        """List all users visible under a role.

        The role must be expanded first.

        :param role_name: The name of the role.
        :returns: List of usernames under the role.
        """
        self._show_users_panel()
        return self.tree.list_names(self.USER_ITEM_PREFIX)
