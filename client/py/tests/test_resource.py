#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest
from pydantic import ValidationError

import synnax as sy


@pytest.mark.resource
class TestID:
    """ID's string form is the wire format the Core parses (e.g. the imex parent
    query param), so it must serialize as type:key."""

    def test_str_is_type_colon_key(self) -> None:
        id = sy.resource.ID(type="project", key="abc-123")
        assert str(id) == "project:abc-123"

    def test_str_round_trips_through_the_parsing_constructor(self) -> None:
        original = sy.resource.ID(type="log", key=str(uuid4()))
        assert sy.resource.ID(str(original)) == original

    def test_copy_constructor(self) -> None:
        original = sy.resource.ID(type="channel", key="12")
        assert sy.resource.ID(original) == original

    def test_invalid_type(self) -> None:
        with pytest.raises(ValidationError, match="Input should be 'arc'"):
            sy.resource.ID("dog:1")
