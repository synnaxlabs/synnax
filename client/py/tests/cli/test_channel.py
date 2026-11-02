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
from synnax.cli.channel import prompt_group_names
from synnax.cli.flow import Context

OPTIONS = ["temp_1", "temp_2", "pressure"]


@pytest.mark.cli
class TestPromptGroupNames:
    def test_groups_by_name_index_and_pattern(self) -> None:
        """Should group the channels that each matcher selects."""
        c = console.Mock(responses=["pressure, 0, temp*"])
        assert prompt_group_names(Context(console=c), OPTIONS) == {
            "pressure": ["pressure"],
            "0": ["temp_1"],
            "temp*": ["temp_1", "temp_2"],
        }

    def test_skips_a_bad_index_when_the_user_continues(self) -> None:
        """Should leave out an index past the end when the user continues."""
        c = console.Mock(responses=["9, pressure", True])
        assert prompt_group_names(Context(console=c), OPTIONS) == {
            "pressure": ["pressure"]
        }

    def test_returns_none_when_the_user_stops_at_a_bad_index(self) -> None:
        """Should return None when the user stops after a bad index."""
        c = console.Mock(responses=["9", False])
        assert prompt_group_names(Context(console=c), OPTIONS) is None

    def test_returns_none_when_the_user_stops_at_an_unmatched_pattern(self) -> None:
        """Should return None when the user stops after a pattern that matches
        nothing."""
        c = console.Mock(responses=["flow*", False])
        assert prompt_group_names(Context(console=c), OPTIONS) is None
