#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

import pytest
from pydantic import TypeAdapter, ValidationError

import synnax as sy

FRAMING = TypeAdapter(sy.bus.Framing)


@pytest.mark.bus
class TestFraming:
    """Tests for the framing of a bus task."""

    def test_length_size_round_trip(self) -> None:
        """Should read and write a length size as its schema value."""
        framing = FRAMING.validate_json(
            '{"type": "sync", "sync": "qg==", "length_size": 4}'
        )
        assert isinstance(framing, sy.bus.SyncFraming)
        assert framing.length_size is sy.bus.LengthSize.four
        assert '"length_size":4' in framing.model_dump_json()

    @pytest.mark.parametrize(
        ("data", "loc"),
        [
            ({"type": "sync", "sync": ""}, ("sync", "sync")),
            (
                {"type": "sync", "sync": "qg==", "length_size": 3},
                ("sync", "length_size"),
            ),
            ({"type": "sync", "sync": "qg==", "max_length": 0}, ("sync", "max_length")),
            (
                {"type": "sync", "sync": "qg==", "checksum": {"algorithm": "crc8"}},
                ("sync", "checksum", "algorithm"),
            ),
            ({"type": "fixed", "length": 0}, ("fixed", "length")),
            ({"type": "fixed", "length": 65537}, ("fixed", "length")),
            ({"type": "delimited", "delimiter": ""}, ("delimited", "delimiter")),
        ],
    )
    def test_refuse_invalid_field(
        self, data: dict[str, object], loc: tuple[str, ...]
    ) -> None:
        """Should refuse an invalid framing at the path of its field."""
        with pytest.raises(ValidationError) as exc:
            FRAMING.validate_json(json.dumps(data))
        assert [e["loc"] for e in exc.value.errors()] == [loc]


@pytest.mark.bus
class TestPoll:
    """Tests for the poll of a bus task."""

    @pytest.mark.parametrize(
        ("data", "loc"),
        [
            ({"query": "", "rate": 1}, ("query",)),
            ({"query": "qg==", "rate": 0}, ("rate",)),
        ],
    )
    def test_refuse_invalid_field(
        self, data: dict[str, object], loc: tuple[str, ...]
    ) -> None:
        """Should refuse an invalid poll at the path of its field."""
        with pytest.raises(ValidationError) as exc:
            sy.bus.Poll.model_validate_json(json.dumps(data))
        assert [e["loc"] for e in exc.value.errors()] == [loc]
