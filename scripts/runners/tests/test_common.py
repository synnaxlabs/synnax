#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from common import instance_for, instance_prefix


class TestInstanceFor:
    """Tests for matching a runner to its EC2 instance."""

    def test_should_return_the_instance_the_suffix_names(self) -> None:
        found: dict[str, dict] = {"i-0f44aa": {}, "i-0b12cc": {}}
        assert instance_for("ubuntu-build-bot-i-0f44", found) == "i-0f44aa"

    def test_should_return_none_without_a_match(self) -> None:
        assert instance_for("ubuntu-build-bot-i-0f44", {"i-0b12cc": {}}) is None

    def test_should_raise_when_the_suffix_is_ambiguous(self) -> None:
        found: dict[str, dict] = {"i-0f44aa": {}, "i-0f44bb": {}}
        with pytest.raises(LookupError):
            instance_for("ubuntu-build-bot-i-0f44", found)

    def test_should_ignore_runners_without_an_instance(self) -> None:
        assert instance_prefix("github-hosted") is None
