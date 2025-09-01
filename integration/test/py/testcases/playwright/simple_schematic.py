#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
import re
from testcases.playwright.playwright import Playwright

class Simple_Schematic(Playwright):
    """
    Open all pages in the "New Component" window and close them
    """
    def setup(self) -> None:

        super().setup()
        self.open_page("Schematic")
        
        
        #self.page.locator(".pluto-resize__handle.pluto--bordered-top").click()
        
        #new_component = self.page.locator("[data-testid^='psi']").last()
        #new_component.wait_for
        # test_id = new_component.get_attribute("data-testid")
        #print(test_id)
        #self.page.get_by_test_id(test_id).locator("div").filter(has_text="psi").first.click()
        #self.page.get_by_text("Telemetry").click()
        #self.page.get_by_role("button", name="pluto-icon--channel Select a").click()
        #page.get_by_text("test_conductor_time").click()
        #page.locator(".pluto-resize__handle.pluto--bordered-top").click()

    def _get_node(self, node_id: str):
        if not node_id:
            raise ValueError("node_id cannot be empty")
        node = self.page.get_by_test_id(node_id)
        node.locator("div").first.click()
        time.sleep(0.1)
        return node
    
    def add_schematic_value(
        self, 
        channel_name: str, 
        notation=None, 
        precision=None, 
        averaging_window=None, 
        staleness_color: str = None,
        staleness_timeout=None,
    ) -> str:
        
        if channel_name.strip() == "":
            raise ValueError("Channel name cannot be empty")
        
        # Validate hex color if provided
        if staleness_color is not None:
            if not re.match(r'^#[0-9A-Fa-f]{6}$', staleness_color):
                raise ValueError("staleness_color must be a valid hex color (e.g., #FF5733)")

        # Count existing nodes before adding
        nodes_count = len(self.page.locator("[data-testid^='rf__node-']").all())

        # Add new component. Assume we are already on a schematic page.
        self.page.locator(".react-flow__pane").dblclick()
        self.page.get_by_text("Value").click()
        
        # Wait for new node to appear
        self.page.wait_for_function(f"document.querySelectorAll('[data-testid^=\"rf__node-\"]').length > {nodes_count}")
        
        # Get all nodes and find the new one
        all_nodes = self.page.locator("[data-testid^='rf__node-']").all()
        node_id = all_nodes[-1].get_attribute("data-testid")  # Last one should be the newest
    
        # Configure the new node
        self.page.get_by_test_id(node_id).locator("div").first.click()
        self.page.get_by_text("Telemetry").click()
        self.page.get_by_role("button", name="pluto-icon--channel Select a").click()
        self.page.get_by_text(channel_name).click()

        if notation is not None:
            if notation.lower() == "scientific":
                self.page.get_by_text("Scientific").click()
            elif notation.lower() == "engineering":
                self.page.get_by_text("Engineering").click()
            elif notation.lower() == "standard":
                self.page.get_by_text("Standard").click()

        if precision is not None:
            precision_input = self.page.locator("text=Precision").locator("..").locator("input")
            precision_input.fill(str(precision))
            precision_input.press("Enter")

        if averaging_window is not None:
            averaging_window_input = self.page.locator("text=Averaging Window").locator("..").locator("input")
            averaging_window_input.fill(str(averaging_window))
            averaging_window_input.press("Enter")

        if staleness_color is not None:
            color_button = self.page.locator("text=Color").locator("..").locator("button")
            color_button.click()
            hex_input = self.page.locator("text=Hex").locator("..").locator("input")
            hex_input.fill(staleness_color.replace("#", ""))  # Remove # since it might be auto-added
            hex_input.press("Enter")
            self.page.keyboard.press("Escape")
            
        
        if staleness_timeout is not None:
            staleness_timeout_input = self.page.locator("text=Staleness Timeout").locator("..").locator("input")
            staleness_timeout_input.fill(str(staleness_timeout))
            staleness_timeout_input.press("Enter")

        return node_id
        
    def get_schematic_value_props(self, node_id: str) -> dict:
        """
        Get properties of a schematic value node by its ID
        Returns a dictionary with the node's properties
        """
        self._get_node(node_id)
        
        # Extract properties - adjust selectors based on actual UI structure
        props = {
            "channel": "",
            "notation": "",
            "precision": -1,
            "averaging_window": -1,
            "staleness_color": "",
            "staleness_timeout": -1,
        }
        

        # Channel Name
        channel_display = self.page.locator("text=Input Channel").locator("..").locator("button")
        if channel_display.count() > 0:
            props["channel"] = channel_display.inner_text().strip()

        # Precision
        precision_input = self.page.locator("text=Precision").locator("..").locator("input")
        props["precision"] = int(precision_input.input_value())
            
        # Averaging Window
        avg_input = self.page.locator("text=Averaging Window").locator("..").locator("input")
        props["averaging_window"] = int(avg_input.input_value())

            
        # Staleness Timeout
        timeout_input = self.page.locator("text=Staleness Timeout").locator("..").locator("input")
        props["staleness_timeout"] = int(timeout_input.input_value())
            
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
                props["staleness_color"] = f"#{hex_value}" if not hex_value.startswith("#") else hex_value
        # Close color picker
        self.page.keyboard.press("Escape")


        return props

    def edit_schematic_value_props(
        self, 
        node_id: str, 
        channel_name=None, 
        notation=None, 
        precision=None, 
        averaging_window=None, 
        staleness_color: str = None,
        staleness_timeout=None,
    ) -> None:
        """
        Get properties of a schematic value node by its ID
        Returns a dictionary with the node's properties
        """
        self._get_node(node_id)
        
        # Channel Name - click the channel dropdown button
        if channel_name is not None:
            channel_button = self.page.locator("text=Input Channel").locator("..").locator("button").first
            channel_button.click()
            self.page.get_by_text(channel_name).click()

        if notation is not None:
            notation_button = self.page.locator("text=Notation").locator("..").locator("button").first
            notation_button.click()
            self.page.get_by_text(notation).click()

        if precision is not None:
            precision_input = self.page.locator("text=Precision").locator("..").locator("input")
            precision_input.fill(str(precision))
            precision_input.press("Enter")

        if averaging_window is not None:
            averaging_window_input = self.page.locator("text=Averaging Window").locator("..").locator("input")
            averaging_window_input.fill(str(averaging_window))
            averaging_window_input.press("Enter")

        if staleness_color is not None:
            color_button = self.page.locator("text=Color").locator("..").locator("button")
            color_button.click()
            hex_input = self.page.locator("text=Hex").locator("..").locator("input")
            hex_input.fill(staleness_color.replace("#", ""))  # Remove # since it might be auto-added
            hex_input.press("Enter")
            self.page.keyboard.press("Escape")

        if staleness_timeout is not None:
            staleness_timeout_input = self.page.locator("text=Staleness Timeout").locator("..").locator("input")
            staleness_timeout_input.fill(str(staleness_timeout))
            staleness_timeout_input.press("Enter")

    def run(self) -> None:

        self._log_message("Checking default properties of schematic value")
        # Create a schematic value with default properties
        node_id = self.add_schematic_value("simple_schematic_uptime")
        default_props = self.get_schematic_value_props(node_id)

        expected_default_props = {
            "channel": "simple_schematic_uptime",
            "notation": "standard",
            "precision": 2,
            "averaging_window": 1,
            "staleness_color": "#C29D0A", #pluto-warning-m1
            "staleness_timeout": 5,
        }

        assert default_props == expected_default_props, f"Props mismatch!\nActual: {default_props}\nExpected: {expected_default_props}"
    

        self._log_message("Checking edited properties of schematic value")
        self.edit_schematic_value_props(node_id, 
            channel_name="simple_schematic_time", 
            notation="scientific", 
            precision=4, 
            averaging_window=4, 
            staleness_color="#FF0000", 
            staleness_timeout=10
        )
        expected_edited_props = {
            "channel": "simple_schematic_time",
            "notation": "scientific",
            "precision": 4,
            "averaging_window": 4,
            "staleness_color": "#FF0000", #pluto-warning-m1
            "staleness_timeout": 10,
        }
        edited_props = self.get_schematic_value_props(node_id)
        assert edited_props == expected_edited_props, f"Props mismatch!\nActual: {edited_props}\nExpected: {expected_edited_props}"
        

        time.sleep(10)