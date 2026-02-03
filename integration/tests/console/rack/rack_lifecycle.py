#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import synnax as sy

from console.case import ConsoleCase


class RackLifecycle(ConsoleCase):
    """Test the lifecycle of racks in the Console UI."""

    def setup(self) -> None:
        super().setup()
        self.rand_suffix: int = random.randint(1000, 9999)
        self.rack_name: str = f"TestRack_{self.rand_suffix}"
        self.test_rack: sy.Rack = self.client.racks.create(name=self.rack_name)

    def run(self) -> None:
        """Run all rack lifecycle tests."""
        self.test_rack_visible_in_toolbar()
        self.test_rack_status_display()
        self.test_rename_rack()
        self.test_copy_rack_key()
        self.test_delete_rack()

    def test_rack_visible_in_toolbar(self) -> None:
        """Test that a rack appears in the devices toolbar."""
        self.log("Testing: Rack visible in devices toolbar")
        exists = self.console.rack.exists(self.rack_name)
        assert exists, f"Rack '{self.rack_name}' should be visible in devices toolbar"

    def test_rack_status_display(self) -> None:
        """Test that rack status is displayed correctly."""
        self.log("Testing: Rack status display")
        status = self.console.rack.get_status(self.rack_name)
        assert status["variant"] in [
            "disabled",
            "warning",
        ], f"Expected disabled/warning status without driver, got {status['variant']}"

    def test_rename_rack(self) -> None:
        """Test renaming a rack via context menu."""
        self.log("Testing: Rename rack")
        new_name = f"RenamedRack_{self.rand_suffix}"
        self.console.rack.rename(old_name=self.rack_name, new_name=new_name)
        self.console.rack.rename(old_name=new_name, new_name=self.rack_name)

    def test_copy_rack_key(self) -> None:
        """Test copying rack key via context menu."""
        self.log("Testing: Copy rack key")
        rack_key = self.console.rack.copy_key(self.rack_name)
        assert rack_key, "Should have extracted rack key from element"
        assert rack_key == str(
            self.test_rack.key
        ), f"Key should match: {rack_key} vs {self.test_rack.key}"

    def test_delete_rack(self) -> None:
        """Test deleting a rack via context menu."""
        self.log("Testing: Delete rack")
        delete_name = f"RackToDelete_{self.rand_suffix}"
        self.client.racks.create(name=delete_name)
        self.page.keyboard.press("d")
        self.page.keyboard.press("d")
        assert self.console.rack.exists(
            delete_name
        ), f"Rack '{delete_name}' should exist before deletion"
        self.console.rack.delete(delete_name)
