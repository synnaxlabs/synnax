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
from testcases.playwright.schematic import Schematic


class Schematic_Staleness(Schematic):
    """
    Verify staleness behavior of schematic value nodes
    This test must run alongside "schematic_staleness_producer.py"
    """

    def run(self) -> None:

        time.sleep(10)
