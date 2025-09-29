#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

from console.case import ConsoleCase


class Snap_Pages_To_Grid(ConsoleCase):
    """
    Open all pages in the "New Component" window and close them
    """

    def run(self) -> None:
        """
        Snap pages and take a screenshot
        """
        console = self.console
        console.schematic.new()
        console.plot.new()
        console.log.new()

        console.log.move("top")
        console.plot.move("right")
        console.schematic.move("bottom")

        console.screenshot()
