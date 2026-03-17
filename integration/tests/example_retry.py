#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import tempfile
from pathlib import Path

from framework.test_case import TestCase


class RetryExample(TestCase):
    """Fails on the first attempt, passes on the second.

    Uses a temp file as a marker — if the file doesn't exist, creates it and
    fails. On the next run the file is found, removed, and the test passes.
    """

    def setup(self) -> None:
        self._marker = Path(tempfile.gettempdir()) / "synnax_retry_example"
        super().setup()

    def run(self) -> None:
        if not self._marker.exists():
            self._marker.touch()
            self.log("First attempt — marker created, failing intentionally")
            raise Exception("Intentional first-attempt failure")

        self._marker.unlink()
        self.log("Second attempt — marker found and removed, passing")
