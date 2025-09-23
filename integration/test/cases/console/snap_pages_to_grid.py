#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from test.console.console_case import ConsoleCase
import time

class Snap_Pages_To_Grid(ConsoleCase):
    """
    Open all pages in the "New Component" window and close them
    """

    def run(self) -> None:
        """
        Open and close pages in 2 ways
        """
        console = self.console

        console.schematic.new()
        console.plot.new()
        console.table.new()
        console.log.new()



        console.schematic.move("top")
        console.table.move("right")

        time.sleep(5)