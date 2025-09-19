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
import inspect
from test.console.console import Console
from typing import Any, Dict, Optional
from abc import ABC
from playwright.sync_api import Page


class SchematicNode(ABC):
    """Base class for all schematic nodes"""

    def __init__(self, page: Page, node_id: str, channel_name: str):
        self.page = page
        self.node_id = node_id
        self.channel_name = channel_name

        self.set_label(channel_name)


    def _click_node(self) -> Any:
        node = self.page.get_by_test_id(self.node_id)
        node.locator("div").first.click()
        time.sleep(0.1)

    def set_label(self, label: str) -> None:
        self._click_node()

        self.page.get_by_text("Style").click()
        label_input = (
                self.page.locator("text=Label").locator("..").locator("input").first
            )
        label_input.fill(label)
        
        
    def set_channel(self, input_field: str, channel_name: str) -> None:

        if channel_name is not None:
            channel_button = (
                self.page.locator(f"text={input_field}")
                .locator("..")
                .locator("button")
                .first
            )
            channel_button.click()
            search_input = self.page.locator("input[placeholder*='Search']")
            search_input.fill(channel_name)
            
            # Using keyboard navigation for selection to avoid conflicts
            # with other page elements (such as the node label).
            search_input.press("Control+a")
            search_input.type(channel_name)
            self.page.wait_for_timeout(300)
            search_input.press("ArrowDown")
            search_input.press("Enter")


    def click(self):
        """Click on the node"""
        return self._click_node()


class ValueNode(SchematicNode):
    """Schematic value/telemetry node"""

    notation: str
    precision: int
    averaging_window: int
    stale_color: str
    stale_timeout: int

    def __init__(self, page: Page, node_id: str, channel_name: str):
        super().__init__(page, node_id, channel_name)

    def edit_properties(self,
                       channel_name: Optional[str] = None,
                       properties: Optional[Dict[str, Any]] = None,
                       # Backward compatibility - individual parameters
                       notation: Optional[str] = None,
                       precision: Optional[str] = None,
                       averaging_window: Optional[str] = None,
                       stale_color: Optional[str] = None,
                       stale_timeout: Optional[str] = None) -> None:
        if properties:
            notation = properties.get("notation", notation)
            precision = properties.get("precision", precision)
            averaging_window = properties.get("averaging_window", averaging_window)
            stale_color = properties.get("stale_color", stale_color)
            stale_timeout = properties.get("stale_timeout", stale_timeout)

        self._click_node()
        self.page.wait_for_selector("text=Properties", timeout=3000)
        self.page.get_by_text("Properties").click()
        self.page.wait_for_selector("text=Telemetry", timeout=3000)
        self.page.get_by_text("Telemetry").click()

        self.page.wait_for_selector("text=Input Channel", timeout=3000)
        self.set_channel("Input Channel", channel_name)

        if notation is not None:
            notation_button = (
                self.page.locator("text=Notation").locator("..").locator("button").first
            )
            notation_button.click()
            self.page.get_by_text(notation).click()

        if precision is not None:
            precision_input = (
                self.page.locator("text=Precision").locator("..").locator("input")
            )
            precision_input.fill(str(precision))
            precision_input.press("Enter")

        if averaging_window is not None:
            averaging_window_input = (
                self.page.locator("text=Averaging Window")
                .locator("..")
                .locator("input")
            )
            averaging_window_input.fill(str(averaging_window))
            averaging_window_input.press("Enter")

        if stale_color is not None:
            color_button = (
                self.page.locator("text=Color").locator("..").locator("button")
            )
            color_button.click()
            hex_input = self.page.locator("text=Hex").locator("..").locator("input")
            hex_input.fill(
                stale_color.replace("#", "")
            )  # Remove # since it might be auto-added
            hex_input.press("Enter")
            self.page.keyboard.press("Escape")

        if stale_timeout is not None:
            stale_timeout_input = (
                self.page.locator("text=Stale Timeout").locator("..").locator("input")
            )
            stale_timeout_input.fill(str(stale_timeout))
            stale_timeout_input.press("Enter")

    def get_value(self) -> float:
        """Get the current value of the node"""
        self._click_node()
        value_str = (
            self.page.locator("text=Value").locator("..").locator("input").input_value()
        )
        return float(value_str)

    def get_properties(self) -> Dict[str, Any]:
        """Get the current properties of the node"""
        self._click_node()

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


class SetpointNode(SchematicNode):
    """Schematic setpoint/control node"""

    def __init__(self, page: Page, node_id: str, channel_name: str):
        super().__init__(page, node_id, channel_name)

    def edit_properties(self, channel_name: Optional[str] = None) -> None:


        self._click_node()
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Control").last.click()

        self.set_channel("Command Channel", channel_name)
        


    def set_control_authority(self, authority: int) -> None:
        self._click_node()
        # Not to be confused with the "Properties>Control" 
        self.page.get_by_text("Control").first.click()

        control_authority_input = (
            self.page.locator("text=Control Authority")
            .locator("..")
            .locator("input")
        )
        control_authority_input.fill(authority)
        control_authority_input.press("Enter")


class Schematic(Console):
    """
    Parent class for schematic tests
    """

    def setup(self) -> None:
        super().setup()
        self.create_page("Schematic")

    def create_node(self, node_type: str, node_id: str, channel_name: str, **kwargs) -> SchematicNode:
        """Factory method to create node objects"""
        if node_type.lower() == "value":
            return ValueNode(self.page, node_id, channel_name, **kwargs)
        elif node_type.lower() == "setpoint":
            return SetpointNode(self.page, node_id, channel_name, **kwargs)
        else:
            raise ValueError(f"Unknown node type: {node_type}")

    def _get_node(self, node_id: str) -> Any:
        if not node_id:
            raise ValueError("node_id cannot be empty")
        node = self.page.get_by_test_id(node_id)
        node.locator("div").first.click()
        time.sleep(0.1)
        return node

    def add_to_schematic(
        self,
        node_type: str,
        channel_name: str,
        **kwargs
    ) -> SchematicNode:
        """
        Add a node to the schematic and return the configured node object

        Args:
            node_type: Type of node ("Value", "Setpoint", etc.)
            channel_name: Channel name for the node
            **kwargs: Node-specific configuration parameters

        Returns:
            Configured SchematicNode object
        """
        if channel_name.strip() == "":
            raise ValueError("Channel name cannot be empty")

        # Validate hex color if provided
        if 'stale_color' in kwargs and kwargs['stale_color'] is not None:
            if not re.match(r"^#[0-9A-Fa-f]{6}$", kwargs['stale_color']):
                raise ValueError(
                    "stale_color must be a valid hex color (e.g., #FF5733)"
                )

        # Count existing nodes before adding
        nodes_count = len(self.page.locator("[data-testid^='rf__node-']").all())

        self.click_on_pane()
        self.page.get_by_text(node_type).first.click()

        # Wait for new node to appear
        self.page.wait_for_function(
            f"document.querySelectorAll('[data-testid^=\"rf__node-\"]').length > {nodes_count}"
        )

        # Get all nodes and find the new one
        all_nodes = self.page.locator("[data-testid^='rf__node-']").all()
        node_id = (
            all_nodes[-1].get_attribute("data-testid") or "unknown"
        )  # Last one should be the newest

        # Create SchematicNode object from factory
        node = self.create_node(node_type, node_id, channel_name, **kwargs)

        # Set up the node with channel and any provided properties
        if node_type.lower() == "value":
            # Apply kwargs as properties for ValueNode
            node.edit_properties(channel_name, kwargs)
        elif node_type.lower() == "setpoint":
            # Apply kwargs as properties for SetpointNode
            node.edit_properties(channel_name)
        
        
        return node

    def click_on_pane(self) -> None:

        # Going to change how this is done.
        # Purpose is to click off of the node.
        # MIGHT to move this into the node functionality
        self.page.wait_for_selector(".react-flow__pane", timeout=5000)
        self.page.locator(".react-flow__pane").dblclick()
        pane = self.page.locator(".react-flow__pane")
        box = pane.bounding_box()
        # Click in the center of the pane
        x = box["x"] + box["width"] * 0.95
        y = box["y"] + box["height"] * 0.95
        self.page.mouse.dblclick(x, y)

