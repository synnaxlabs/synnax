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
from typing import Any, Dict, List, Optional, Union

from testcases.playwright.playwright import Playwright


class Plot(Playwright):
    """
    Parent class for Plot tests
    """

    DATA: Dict[str, Any] = {
        "Y1": [],
        "Y2": [],
        "Ranges": [],
        "X1": None,
    }

    def setup(self) -> None:
        super().setup()

        self.create_page("Line Plot", f"{self.name}")
        self.page.locator(".pluto--no-select").dblclick()

    def sub_Y1(self, channel_ids: Union[str, List[str]]) -> None:
        self.subscribe(channel_ids)

    def sub_Y2(self, channel_ids: Union[str, List[str]]) -> None:
        self.subscribe(channel_ids)

    def add_Y1(self, channel_ids: Union[str, List[str]]) -> None:
        self.add_Y("Y1", channel_ids)

    def add_Y2(self, channel_ids: Union[str, List[str]]) -> None:
        self.add_Y("Y2", channel_ids)

    def add_Y(self, axis: str, channel_ids: Union[str, List[str]]) -> None:

        if axis != "Y1" and axis != "Y2":
            raise ValueError(f"Invalid axis: {axis}")

        # Close any open dialogs first
        self.page.keyboard.press("Escape")
        time.sleep(0.2)

        # Use a more direct selector for the channel selection button
        selector = self.page.get_by_text(f"{axis} Select Channels", exact=True)
        selector.click(timeout=5000)

        # Wait for the channel list to be visible
        self.page.wait_for_selector(".pluto-list__item", timeout=3000)

        # Handle both string and list inputs
        channels = channel_ids if isinstance(channel_ids, list) else [channel_ids]
        for channel in channels:
            # Clear search box first to ensure all channels are visible
            search_box = self.page.locator("input[placeholder*='Search']")
            if search_box.count() > 0:
                search_box.clear()
                search_box.fill(channel)
                time.sleep(0.1)  # Wait for filtering

            # Use first() to handle multiple matches (like t_c matching test_conductor_test_case_count)
            channel_element = self.page.locator(
                f".pluto-list__item:has-text('{channel}')"
            ).first
            # Wait for the specific channel to be visible and scroll into view
            channel_element.wait_for(state="visible", timeout=5000)
            channel_element.scroll_into_view_if_needed()
            channel_element.click()
            self.DATA[axis].append(channel)

        # Clear search box after selection
        search_box = self.page.locator("input[placeholder*='Search']")
        if search_box.count() > 0:
            search_box.clear()

        self.page.keyboard.press("Escape")

    def add_ranges(self, ranges: list[str]) -> None:
        Range_Options = ["30s", "1m", "5m", "15m", "30m"]

        self.page.get_by_text("Select Ranges").click()
        for range in ranges:
            if range in Range_Options and range not in self.DATA["Ranges"]:
                self.page.get_by_text(range, exact=True).click()
                self.DATA["Ranges"].append(range)
        self.ESCAPE

    def save_screenshot(self, path: Optional[str] = None) -> None:
        """
        Save a screenshot of the plot area including axes with margin
        """
        if path is None:
            path = f"{self.name}.png"

        plot_locator = self.page.locator(".pluto-line-plot")

        # Get the bounding box and add margin
        box = plot_locator.bounding_box()
        if box:
            margin = 10
            clip_area = {
                "x": max(0, box["x"] - margin),
                "y": max(0, box["y"] - margin),
                "width": box["width"] + 2 * margin,
                "height": box["height"] + 2 * margin,
            }

            self.page.screenshot(
                path=path,
                clip=clip_area,
                animations="disabled",
                omit_background=False,
                type="png",
                scale="device",
            )
        else:
            # Fallback to element screenshot if bounding box fails
            plot_locator.screenshot(
                path=path,
                animations="disabled",
                omit_background=False,
                type="png",
                scale="device",
            )

    def set_X1_axis(self, config: Dict[str, Any]) -> None:
        self.set_axis("X1", config)

    def set_Y1_axis(self, config: Dict[str, Any]) -> None:
        self.set_axis("Y1", config)

    def set_Y2_axis(self, config: Dict[str, Any]) -> None:
        self.set_axis("Y2", config)

    def set_axis(self, axis: str, config: Dict[str, Any]) -> None:
        self.page.get_by_text("Axes").click()
        self.page.locator(f"#{axis.lower()}").click()

        for key, value in config.items():
            try:
                if key in ["Lower Bound", "Upper Bound", "Tick Spacing", "Label"]:
                    # For text input fields, find the input associated with the label
                    input_field = self.page.locator(
                        f"label:has-text('{key}') + div input"
                    )
                    input_field.clear()
                    input_field.fill(str(value))
                elif key == "Label Direction":
                    # For button groups, click the appropriate button
                    if str(value).lower() == "up":
                        self.page.locator(
                            "label:has-text('Label Direction') + div button:has([aria-label='pluto-icon--arrow-up'])"
                        ).click()
                    else:  # right/left/horizontal
                        self.page.locator(
                            "label:has-text('Label Direction') + div button:has([aria-label='pluto-icon--arrow-right'])"
                        ).click()
                elif key == "Label Size":
                    # For size buttons, click the appropriate size
                    size_mapping = {
                        "xs": "XS",
                        "s": "S",
                        "m": "M",
                        "l": "L",
                        "xl": "XL",
                    }
                    button_text = size_mapping.get(
                        str(value).lower(), str(value).upper()
                    )
                    self.page.locator(
                        f"label:has-text('Label Size') + div button:has-text('{button_text}')"
                    ).click()
                else:
                    # Fallback for unknown keys
                    self.page.locator(key).fill(str(value))
            except Exception as e:
                raise ValueError(
                    f'Warning: Failed to set axis property "{key}" to "{value}": {e}'
                )
        self.ENTER
