#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from framework.models import STATUS, SYMBOLS


class TestStatus:
    def test_skipped_order(self) -> None:
        """Should let a failure replace a skip, but not a pass."""
        assert STATUS.PASSED.value < STATUS.SKIPPED.value < STATUS.FAILED.value

    def test_skipped_symbol(self) -> None:
        """Should give skipped tests their own symbol."""
        assert SYMBOLS.get_symbol(STATUS.SKIPPED) == SYMBOLS.SKIPPED.value
