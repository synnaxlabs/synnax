#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass

import synnax as sy

from framework.test_case import STATUS


@dataclass
class Test:
    """Data class to store test execution results."""

    test_name: str
    status: STATUS
    name: str | None = None
    error_message: str | None = None
    range: sy.Range | None = None

    def __str__(self) -> str:
        """Return display name for test result."""
        if self.name and self.name != self.test_name.split("/")[-1]:
            return f"{self.test_name} ({self.name})"
        return self.test_name
