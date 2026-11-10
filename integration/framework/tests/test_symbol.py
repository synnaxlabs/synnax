#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from console.schematic.symbol import option_text


class TestOptionText:
    @pytest.mark.parametrize(
        ("value", "text"),
        [(1.0, "1"), (1.5, "1.5"), (1234567.0, "1234567"), (0.1234567, "0.1234567")],
    )
    def test_keeps_every_digit(self, value: float, text: str) -> None:
        """Should keep each digit of a value with more than six significant digits."""
        assert option_text(value) == text
        assert float(option_text(value)) == value
