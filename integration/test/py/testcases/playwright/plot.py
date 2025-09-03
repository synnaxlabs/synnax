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


class Plot(Playwright):
    """
    Parent class for Plot tests
    """

    def setup(self) -> None:
        super().setup()
        self.create_page("Line Plot")

    def get_latest_value(self, node_id: str) -> float:
        """
        Return the latest value of a channel in the plot
        """

        return None
