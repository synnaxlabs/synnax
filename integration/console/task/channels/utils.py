#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re

from playwright.sync_api import Locator

from console.layout import LayoutClient


def is_numeric_string(value: str | bool) -> bool:
    """Check if a string represents a numeric value."""
    if not isinstance(value, str):
        return False

    value = value.strip()
    if not value:
        return False

    if value.startswith("-"):
        value = value[1:]

    parts = value.split(".")
    if len(parts) > 2:
        return False

    return all(part.isdigit() for part in parts if part)


def _port_grid(layout: LayoutClient) -> Locator:
    return layout.page.get_by_role("group", name="Port", exact=True)


def set_port(layout: LayoutClient, port: int) -> None:
    """Set the selected channel's port.

    A device with a port list shows its ports as a grid of buttons named by port, such
    as "ai3". A device without one shows a number field.
    """
    grid = _port_grid(layout)
    if grid.count() == 0:
        layout.fill_input_field("Port", str(port))
        return
    grid.get_by_role("button", name=re.compile(rf"^[a-z]+{port}$")).click()


def get_port(layout: LayoutClient) -> str:
    """Get the selected channel's port from the port grid or the number field."""
    grid = _port_grid(layout)
    if grid.count() == 0:
        return layout.get_input_field("Port")
    name = grid.locator("button[aria-pressed='true']").first.get_attribute("aria-label")
    return re.sub(r"^[a-z]+", "", name or "")
