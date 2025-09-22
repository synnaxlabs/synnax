#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
import time
from abc import ABC, abstractmethod
from test.console.console import Console
from typing import Any, Dict, Optional, Tuple

from playwright.sync_api import Locator, Page


class SchematicSymbol(ABC):
    """Base class for all schematic nodes"""

    page: Page
    node: Locator
    node_id: str
    channel_name: str
    label: str

    def __init__(self, page: Page, node_id: str, channel_name: str):
        self.page = page
        self.node_id = node_id
        self.channel_name = channel_name
        self.label = channel_name

        self.node = self.page.get_by_test_id(self.node_id)
        self.set_label(channel_name)

    def _disable_edit_mode(self) -> None:
        edit_off_icon = self.page.get_by_label("pluto-icon--edit-off")
        if edit_off_icon.count() > 0:
            edit_off_icon.click()

    def _click_node(self) -> None:
        self.node.click()
        time.sleep(0.1)

    def set_label(self, label: str) -> None:
        self._click_node()
        self.page.get_by_text("Style").click()
        label_input = (
            self.page.locator("text=Label").locator("..").locator("input").first
        )
        label_input.fill(label)
        self.label = label

    @abstractmethod
    def edit_properties(
        self,
        channel_name: Optional[str] = None,
        properties: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Edit node properties. Must be implemented by all child classes.

        Args:
            channel_name: Optional channel name to set
            properties: Optional dictionary of properties to set

        Returns:
            Dictionary of applied properties
        """
        pass

    def set_channel(self, input_field: str, channel_name: str) -> None:
        if channel_name is not None:
            channel_button = (
                self.page.locator(f"text={input_field}")
                .locator("..")
                .locator("button")
                .first
            )
            # Click on the selector and fille channel_name
            channel_button.click()
            search_input = self.page.locator("input[placeholder*='Search']")
            search_input.click()
            search_input.fill(channel_name)
            self.page.wait_for_timeout(500)

            # Iterate through dropdown items
            channel_found = False
            item_selector = self.page.locator(".pluto-list__item").all()
            for item in item_selector:
                if item.is_visible() and channel_name in item.inner_text().strip():
                    item.click()
                    channel_found = True
                    break

            if not channel_found:
                raise RuntimeError(
                    f"Could not find channel '{channel_name}' in dropdown"
                )

    def get_properties(self) -> Dict[str, Any]:
        return {}

    def move(self, delta_x: int, delta_y: int) -> None:
        """Move the node by the specified number of pixels using drag"""
        box = self.node.bounding_box()
        if not box:
            raise RuntimeError(f"Could not get bounding box for node {self.node_id}")

        # Calculate target position
        start_x = box["x"] + box["width"] / 2
        start_y = box["y"] + box["height"] / 2
        target_x = start_x + delta_x
        target_y = start_y + delta_y

        # Move
        self.page.mouse.move(start_x, start_y)
        self.page.mouse.down()
        self.page.mouse.move(target_x, target_y, steps=10)
        self.page.mouse.up()

        # Verify the move
        new_box = self.node.bounding_box()
        if not new_box:
            raise RuntimeError(
                f"Could not get new bounding box for node {self.node_id}"
            )

        final_x = new_box["x"] + new_box["width"] / 2
        final_y = new_box["y"] + new_box["height"] / 2

        grid_tolerance = 25
        if (
            abs(final_x - target_x) > grid_tolerance
            or abs(final_y - target_y) > grid_tolerance
        ):
            raise RuntimeError(
                f"Node {self.node_id} moved to ({final_x}, {final_y}) instead of ({target_x}, {target_y})"
            )

    def set_value(self, value: Any = None) -> None:

        if value is None:
            raise ValueError(f"{self.label}: Set Value cannot be None")

        self._disable_edit_mode()
        self._click_node()

        # Fill the input and set the value
        value_input = self.node.locator("input[type='number'], input").first
        value_input.fill(str(value))
        set_button = self.node.locator("button").filter(has_text="Set")
        set_button.click()


class ValueNode(SchematicSymbol):
    """Schematic value/telemetry node"""

    channel_name: str
    notation: str
    precision: int
    averaging_window: int
    stale_color: str
    stale_timeout: int

    def __init__(self, page: Page, node_id: str, channel_name: str):
        super().__init__(page, node_id, channel_name)

    def edit_properties(
        self,
        channel_name: Optional[str] = None,
        properties: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Edit ValueNode properties including channel and telemetry settings."""
        properties = properties or {}

        # Always enforce label = channel_name for easy identification
        if channel_name is not None:
            self.set_label(channel_name)

        # Navigate to Properties > Telemetry tab
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Telemetry").click()

        if channel_name is not None:
            self.set_channel("Input Channel", channel_name)

        if properties.get("notation") is not None:
            notation_button = (
                self.page.locator("text=Notation").locator("..").locator("button").first
            )
            notation_button.click()
            self.page.get_by_text(properties["notation"]).click()

        if properties.get("precision") is not None:
            precision_input = (
                self.page.locator("text=Precision").locator("..").locator("input")
            )
            precision_input.fill(str(properties["precision"]))
            precision_input.press("Enter")

        if properties.get("averaging_window") is not None:
            averaging_window_input = (
                self.page.locator("text=Averaging Window")
                .locator("..")
                .locator("input")
            )
            averaging_window_input.fill(str(properties["averaging_window"]))
            averaging_window_input.press("Enter")

        if properties.get("stale_color") is not None:
            stale_color = properties["stale_color"]
            if not re.match(r"^#[0-9A-Fa-f]{6}$", stale_color):
                raise ValueError(
                    "stale_color must be a valid hex color (e.g., #FF5733)"
                )

            color_button = (
                self.page.locator("text=Color").locator("..").locator("button")
            )
            color_button.click()
            hex_input = self.page.locator("text=Hex").locator("..").locator("input")
            hex_input.fill(stale_color.replace("#", ""))
            hex_input.press("Enter")
            self.page.keyboard.press("Escape")

        if properties.get("stale_timeout") is not None:
            stale_timeout_input = (
                self.page.locator("text=Stale Timeout").locator("..").locator("input")
            )
            stale_timeout_input.fill(str(properties["stale_timeout"]))
            stale_timeout_input.press("Enter")

        return properties

    def get_properties(self) -> Dict[str, Any]:
        """Get the current properties of the node"""
        self._click_node()
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Telemetry").click()

        # Extract properties - adjust selectors based on actual UI structure
        props = {
            "channel": "",
            "notation": "",
            "precision": -1,
            "averaging_window": -1,
            "stale_color": "",
            "stale_timeout": -1,
        }

        # Channel Name
        channel_display = (
            self.page.locator("text=Input Channel").locator("..").locator("button")
        )
        if channel_display.count() > 0:
            props["channel"] = channel_display.inner_text().strip()

        # Precision
        precision_input = (
            self.page.locator("text=Precision").locator("..").locator("input")
        )
        props["precision"] = int(precision_input.input_value())

        # Averaging Window
        avg_input = (
            self.page.locator("text=Averaging Window").locator("..").locator("input")
        )
        props["averaging_window"] = int(avg_input.input_value())

        # Staleness Timeout
        timeout_input = (
            self.page.locator("text=Stale Timeout").locator("..").locator("input")
        )
        props["stale_timeout"] = int(timeout_input.input_value())

        # Notation
        notation_options = ["Scientific", "Engineering", "Standard"]
        for option in notation_options:
            try:
                button = self.page.get_by_text(option).first
                if button.count() > 0:
                    class_name = button.get_attribute("class") or ""
                    if "pluto-btn--filled" in class_name:
                        props["notation"] = str(option).lower()
                        break
            except:
                continue

        # Staleness Color - get hex value from color picker
        color_button = self.page.locator("text=Color").locator("..").locator("button")
        color_button.click()
        hex_input = self.page.locator("text=Hex").locator("..").locator("input")
        if hex_input.count() > 0:
            hex_value = hex_input.input_value()
            if hex_value:
                props["stale_color"] = (
                    f"#{hex_value}" if not hex_value.startswith("#") else hex_value
                )
        # Close color picker
        self.page.keyboard.press("Escape")

        return props


class SetpointNode(SchematicSymbol):
    """Schematic setpoint/control node"""

    def __init__(self, page: Page, node_id: str, channel_name: str):
        super().__init__(page, node_id, channel_name)

    def edit_properties(
        self,
        channel_name: Optional[str] = None,
        properties: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Edit SetpointNode properties including channel settings."""
        properties = properties or {}

        if channel_name is not None:
            self.set_label(channel_name)

            # Navigate to Properties > Control tab
            self.page.get_by_text("Properties").click()
            self.page.get_by_text("Control").last.click()
            self.set_channel("Command Channel", channel_name)

        return properties


class Schematic(Console):
    """
    Parent class for schematic tests
    """

    def setup(self) -> None:
        super().setup()
        self.create_page("Schematic")
        self.page.locator(".react-flow__pane").dblclick()

    def create_node(
        self, node_type: str, node_id: str, channel_name: str
    ) -> SchematicSymbol:
        """Factory method to create node objects"""
        if node_type.lower() == "value":
            return ValueNode(self.page, node_id, channel_name)
        elif node_type.lower() == "setpoint":
            return SetpointNode(self.page, node_id, channel_name)
        else:
            raise ValueError(f"Unknown node type: {node_type}")

    def add_to_schematic(
        self, node_type: str, channel_name: str, properties: Dict[str, Any] = {}
    ) -> SchematicSymbol:
        """
        Add a node to the schematic and return the configured node object
        """
        if channel_name.strip() == "":
            raise ValueError("Channel name cannot be empty")

        # Count existing nodes before adding
        nodes_count = len(self.page.locator("[data-testid^='rf__node-']").all())

        self.click_on_pane()
        self.page.wait_for_selector(f"text={node_type}", timeout=3000)
        self.page.get_by_text(node_type, exact=True).first.click()

        # Wait for new node to appear
        self.page.wait_for_function(
            f"document.querySelectorAll('[data-testid^=\"rf__node-\"]').length > {nodes_count}"
        )

        # Get all nodes and find the new one
        all_nodes = self.page.locator("[data-testid^='rf__node-']").all()
        node_id = (
            all_nodes[-1].get_attribute("data-testid") or "unknown"
        )  # Last one should be the newest

        node = self.create_node(node_type, node_id, channel_name)
        node.edit_properties(channel_name, properties)

        self._log_message(f"Added node {node_type} with channel {channel_name}")

        return node

    def connect_nodes(
        self,
        source_node: SchematicSymbol,
        source_handle: str,
        target_node: SchematicSymbol,
        target_handle: str,
    ) -> None:
        """
        Connect two nodes by dragging from source handle to target handle.
        """
        source_x, source_y = self.find_node_handle(source_node, source_handle)
        target_x, target_y = self.find_node_handle(target_node, target_handle)

        self._log_message(
            f"Connecting {source_node.label}:{source_handle} to {target_node.label}:{target_handle}"
        )

        self.page.mouse.move(source_x, source_y)
        self.page.mouse.down()
        self.page.mouse.move(target_x, target_y, steps=10)
        self.page.mouse.up()

    def find_node_handle(
        self, node: SchematicSymbol, handle: str
    ) -> Tuple[float, float]:
        """Calculate the coordinates of a node's connection handle."""
        node_box = node.node.bounding_box()
        if not node_box:
            raise RuntimeError(f"Could not get bounding box for node {node.label}")

        x, y, w, h = node_box["x"], node_box["y"], node_box["width"], node_box["height"]

        handle_positions = {
            "left": (x, y + h / 2),
            "right": (x + w, y + h / 2),
            "top": (x + w / 2, y),
            "bottom": (x + w / 2, y + h),
        }

        if handle not in handle_positions:
            raise ValueError(
                f"Invalid handle: {handle}. Must be one of {list(handle_positions.keys())}"
            )

        return handle_positions[handle]

    def click_on_pane(self) -> None:

        # Going to change how this is done. Purpose is to click off of
        # the node and onto schematic pane to reset the focus.
        #
        # MIGHT to move this into the node functionality
        self.page.wait_for_selector(".react-flow__pane", timeout=5000)
        self.page.locator(".react-flow__pane").dblclick()
        pane = self.page.locator(".react-flow__pane")
        box = pane.bounding_box()
        if box:
            # Click in the center of the pane
            x = box["x"] + box["width"] * 0.95
            y = box["y"] + box["height"] * 0.95
            self.page.mouse.dblclick(x, y)
