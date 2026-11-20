#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import TypeAdapter, ValidationError

from x import notation

NOTATION = TypeAdapter(notation.Notation)


class TestNotation:
    @pytest.mark.parametrize("value", ["standard", "scientific", "engineering"])
    def test_valid(self, value: str) -> None:
        """Should accept each notation."""
        assert NOTATION.validate_python(value) == value

    def test_invalid(self) -> None:
        """Should reject an unknown notation."""
        with pytest.raises(ValidationError):
            NOTATION.validate_python("roman")
