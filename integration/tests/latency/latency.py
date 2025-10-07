#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import gc

from framework.test_case import TestCase


class Latency(TestCase):
    """
    Latency test setup/teardown
    """

    def setup(self) -> None:
        
        gc.disable()
        self.log("GC Disabled")

    def teardown(self) -> None:
        gc.enable()
        self.log("GC Enabled")
