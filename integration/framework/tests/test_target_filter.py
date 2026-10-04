#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from framework.target_filter import parse_target


class TestParseTarget:
    """Tests for parsing a test conductor target."""

    def test_should_match_any_listed_sequence(self) -> None:
        target = parse_target("console/user,cluster/...")
        assert target.file_filter == ["console"]
        assert target.matches_sequence("user")
        assert target.matches_sequence("cluster")
        assert not target.matches_sequence("pages")

    def test_should_match_every_sequence_without_a_sequence_segment(self) -> None:
        assert parse_target("console").matches_sequence("pages")
