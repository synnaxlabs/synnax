#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from x.dicts import none_to_empty


class TestNoneToEmpty:
    def test_none_returns_empty(self) -> None:
        assert none_to_empty(None) == {}

    def test_populated_returned_unchanged(self) -> None:
        v = {"a": 1, "b": 2}
        assert none_to_empty(v) is v

    def test_empty_returned_unchanged(self) -> None:
        v: dict[str, int] = {}
        assert none_to_empty(v) is v
