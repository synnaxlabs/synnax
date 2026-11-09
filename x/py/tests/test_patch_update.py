#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from typing import Any, TypedDict

import pytest
from pydantic import TypeAdapter

from x import patch


class Settings(TypedDict):
    rate: int
    stream: int


class SettingsUpdate(TypedDict, total=False):
    rate: int
    stream: int


class Holder(TypedDict, total=False):
    name: str
    group: str | None
    settings: patch.Struct[SettingsUpdate, Settings]
    extra: patch.Struct[SettingsUpdate, Settings] | None


HOLDER = TypeAdapter(Holder)


class TestStruct:
    @pytest.mark.parametrize(
        ("value", "data"),
        [
            ({}, "{}"),
            ({"extra": {"stream": 3}}, '{"extra":{"stream":3}}'),
            (
                {"extra": {"set": {"rate": 4, "stream": 0}}},
                '{"extra":{"set":{"rate":4,"stream":0}}}',
            ),
            ({"extra": None}, '{"extra":null}'),
            ({"settings": {"rate": 5}}, '{"settings":{"rate":5}}'),
            ({"group": None}, '{"group":null}'),
        ],
        ids=[
            "absent",
            "merge",
            "replace",
            "clear",
            "required merge",
            "scalar clear",
        ],
    )
    def test_round_trip(self, value: Holder, data: str) -> None:
        """Should encode and decode the JSON form of an update as Go does."""
        assert json.loads(HOLDER.dump_json(value)) == json.loads(data)
        assert HOLDER.validate_json(data) == value


class TestUpdate:
    def test_round_trip(self) -> None:
        """Should carry a partial and its guards."""
        data = (
            '{"value":{"extra":{"set":{"rate":1,"stream":2}}},'
            '"guards":[{"path":["config","type"],"value":"ai"}]}'
        )
        update = patch.Update[Holder].model_validate_json(data)
        assert update.value == {"extra": {"set": {"rate": 1, "stream": 2}}}
        assert update.guards == [patch.Guard(path=["config", "type"], value="ai")]
        assert json.loads(update.model_dump_json()) == json.loads(data)

    def test_guards_default(self) -> None:
        """Should default to no guards."""
        assert patch.Update[Holder].model_validate({"value": {}}).guards == []


class TestList:
    def test_keyed_operators(self) -> None:
        """Should keep an empty set apart from an absent one."""
        ops: Any = {"set": [], "move": [{"key": "a", "after": None}]}
        parsed = patch.List[Settings, str, SettingsUpdate].model_validate(ops)
        assert parsed.set == []
        assert parsed.insert is None
        assert parsed.move == [patch.ListMove[str](key="a")]

    def test_value_operators(self) -> None:
        """Should read value list operators."""
        parsed = patch.ValueList[str].model_validate({"remove": ["a"]})
        assert parsed.remove == ["a"]
        assert parsed.set is None
