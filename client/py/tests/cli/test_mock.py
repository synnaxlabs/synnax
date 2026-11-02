#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from synnax.cli import console
from synnax.cli.console.mock import Output


@pytest.mark.cli
class TestMock:
    def test_default_outputs_separate(self) -> None:
        """Should give each console its own output when none is supplied."""
        first = console.Mock()
        second = console.Mock()
        first.info("hello")
        assert len(first.output.entries) == 1
        assert second.output.entries == []

    def test_supplied_output(self) -> None:
        """Should write to the output the caller supplies."""
        output = Output()
        console.Mock(output=output).info("hello")
        assert [e.message for e in output.entries] == ["hello"]
