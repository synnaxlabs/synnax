#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy


class TestSynnax:
    def test_positional_params(self) -> None:
        """Should refuse connection parameters passed by position."""
        with pytest.raises(TypeError):
            sy.Synnax("localhost", 9090)  # type: ignore[call-arg]
