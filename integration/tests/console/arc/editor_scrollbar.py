#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that the Arc text editor's horizontal scrollbar takes drags."""

import synnax as sy
from console.case import ConsoleCase
from x import random_name

LONG_LINE = "// " + "wide " * 200


class EditorScrollbar(ConsoleCase):
    """Drag the horizontal scrollbar of a text Arc after a pause above it.

    The fade above the task controls overlaps the bottom of the editor. When the fade
    takes pointer events, a pause above the scrollbar makes Monaco hide it, and the
    drag that follows goes to the fade instead.
    """

    arc_name: str

    def setup(self) -> None:
        super().setup()
        self.arc_name = f"editor_scrollbar_{random_name()}"
        self.client.arcs.create(
            name=self.arc_name, text=sy.Text(raw=LONG_LINE), mode="text"
        )

    def teardown(self) -> None:
        self.console.arc.delete(self.arc_name)
        super().teardown()

    def run(self) -> None:
        self.console.arc.open(self.arc_name)
        page = self.page
        # Monaco hides its scrollbars from the accessibility tree, so no role or label
        # reaches the slider. `.editor-scrollable` skips the hover widget scrollbars.
        track = page.locator(
            "[data-mode-id='arc'] .editor-scrollable > .scrollbar.horizontal"
        )
        slider = track.locator(".slider")
        # The slider fills the track until Monaco measures the long line.
        page.wait_for_function(
            "(track) => track.querySelector('.slider').offsetWidth < track.offsetWidth",
            arg=track.element_handle(timeout=5000),
            timeout=5000,
        )
        start = slider.bounding_box()
        assert start is not None, "the horizontal scrollbar has no slider"

        x = start["x"] + start["width"] / 2
        y = start["y"] + start["height"] / 2
        page.mouse.move(x, y - 8)
        sy.sleep(1.0)
        page.mouse.move(x, y)
        page.mouse.down()
        page.mouse.move(x + 200, y, steps=10)
        page.mouse.up()

        end = slider.bounding_box()
        assert end is not None, "the horizontal scrollbar lost its slider"
        assert end["x"] > start["x"] + 50, (
            f"the drag did not move the slider: x went from {start['x']} to {end['x']}"
        )
