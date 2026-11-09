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
from pydantic import BaseModel, TypeAdapter, ValidationError

from x import patch
from x.codec import JSONCodec


class Settings(TypedDict):
    rate: int
    stream: int


class SettingsUpdate(TypedDict, total=False):
    rate: int
    stream: int


class Pair(TypedDict):
    key: str
    rate: int


class PairUpdate(TypedDict, total=False):
    key: str
    rate: int


class Holder(TypedDict, total=False):
    name: str
    group: patch.Change[str]
    settings: patch.Struct[SettingsUpdate, Settings]
    extra: patch.Change[patch.Struct[SettingsUpdate, Settings]]
    tags: patch.ValueListDict[str]
    pairs: patch.ListDict[Pair, str, PairUpdate]


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
            ({"tags": {"set": []}}, '{"tags":{"set":[]}}'),
            (
                {
                    "pairs": {
                        "remove": ["a"],
                        "update": [{"key": "b", "rate": 2}],
                        "move": [{"key": "b"}, {"key": "c", "after": "b"}],
                    }
                },
                '{"pairs":{"remove":["a"],"update":[{"key":"b","rate":2}],'
                '"move":[{"key":"b"},{"key":"c","after":"b"}]}}',
            ),
        ],
        ids=[
            "absent",
            "merge",
            "replace",
            "clear",
            "required merge",
            "scalar clear",
            "empty set",
            "keyed operators",
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

    def test_dict_form_rejects_plain_list(self) -> None:
        """Should reject a plain list where a dict form takes operators."""
        with pytest.raises(ValidationError):
            HOLDER.validate_python({"tags": ["a"]})

    def test_value_operators(self) -> None:
        """Should read value list operators."""
        parsed = patch.ValueList[str].model_validate({"remove": ["a"]})
        assert parsed.remove == ["a"]
        assert parsed.set is None


class Request(BaseModel):
    key: str | None = None
    update: patch.Update[Holder]


class TestClears:
    def test_codec_keeps_clears(self) -> None:
        """Should keep a None in an update value through the client codec."""
        request = Request(update=patch.Update[Holder](value={"group": None}))
        data = JSONCodec().encode(request)
        assert json.loads(data) == {"update": {"value": {"group": None}, "guards": []}}
        assert JSONCodec().decode(data, Request) == request

    def test_nested_clear(self) -> None:
        """Should keep a None below a struct merge and leave absent keys out."""
        value: Holder = {"settings": {"rate": 5}, "extra": None}
        request = Request(update=patch.Update[Holder](value=value))
        assert json.loads(JSONCodec().encode(request))["update"]["value"] == {
            "settings": {"rate": 5},
            "extra": None,
        }
