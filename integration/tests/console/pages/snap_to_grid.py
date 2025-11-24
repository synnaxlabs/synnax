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
from console.log import Log
from console.plot import Plot
from console.schematic.schematic import Schematic


class SnapToGrid(ConsoleCase):
    """
    Open all pages in the "New Component" window and close them
    """

    def run(self) -> None:
        """
        Snap pages and take a screenshot
        """
        console = self.console
        schematic = Schematic(self.client, self.console, "schematic")
        plot = Plot(self.client, self.console, "plot")
        log = Log(self.client, self.console, "log")

        log.move("top")
        plot.move("right")
        schematic.move("bottom")

        console.screenshot()
