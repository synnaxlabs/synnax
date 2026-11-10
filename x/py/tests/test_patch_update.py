#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import warnings
from pathlib import Path
from typing import Any, Literal, TypedDict

import pytest
from pydantic import BaseModel, TypeAdapter, ValidationError

from x import patch
from x.codec import JSONCodec
from x.telem import TimeRange, TimeStamp


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
    cells: patch.TypedMapDict[str, Settings, SettingsUpdate]


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
            (
                {
                    "cells": {
                        "remove": ["c2"],
                        "update": {"c1": {"rate": 5}},
                        "insert": {"c3": {"rate": 1, "stream": 0}},
                    }
                },
                '{"cells":{"remove":["c2"],"update":{"c1":{"rate":5}},'
                '"insert":{"c3":{"rate":1,"stream":0}}}}',
            ),
            ({"cells": {"set": {}}}, '{"cells":{"set":{}}}'),
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
            "map operators",
            "empty map set",
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

    def test_dump_struct_of_custom_types_without_warning(self) -> None:
        class TimeRangeUpdate(TypedDict, total=False):
            start: TimeStamp
            end: TimeStamp

        class Range(TypedDict, total=False):
            time_range: patch.Struct[TimeRangeUpdate, TimeRange]

        with warnings.catch_warnings():
            warnings.simplefilter("error")
            merge = patch.Update[Range](value={"time_range": {"end": TimeStamp(5)}})
            replace = patch.Update[Range](
                value={"time_range": {"set": TimeRange(TimeStamp(1), TimeStamp(5))}}
            )
            assert merge.model_dump_json() == (
                '{"value":{"time_range":{"end":5}},"guards":[]}'
            )
            assert replace.model_dump_json() == (
                '{"value":{"time_range":{"set":{"start":1,"end":5}}},"guards":[]}'
            )

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


class TestTypedMap:
    def test_operators(self) -> None:
        """Should read map operators keyed by map key, and keep an empty set."""
        ops: Any = {"set": {}, "update": {"c1": {"rate": 2}}}
        parsed = patch.TypedMap[str, Settings, SettingsUpdate].model_validate(ops)
        assert parsed.set == {}
        assert parsed.insert is None
        assert parsed.remove is None
        assert parsed.update == {"c1": {"rate": 2}}

    def test_dict_form_rejects_list_operators(self) -> None:
        """Should reject a list where the dict form takes a map."""
        with pytest.raises(ValidationError):
            HOLDER.validate_python({"cells": {"insert": [{"rate": 1, "stream": 0}]}})


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


class AI(TypedDict):
    key: str
    type: Literal["ai"]
    port: int
    range: int
    units: str


class DI(TypedDict):
    key: str
    type: Literal["di"]
    port: int
    line: int


class AIUpdate(TypedDict, total=False):
    key: str
    type: Literal["ai"]
    port: patch.Change[int]
    range: patch.Change[int]
    units: patch.Change[str]


class DIUpdate(TypedDict, total=False):
    key: str
    type: Literal["di"]
    port: patch.Change[int]
    line: patch.Change[int]


class Config(TypedDict):
    rate: int
    channels: list[AI | DI]


class ConfigUpdate(TypedDict, total=False):
    rate: patch.Change[int]
    channels: patch.ListDict[AI | DI, str, AIUpdate | DIUpdate]


class Limits(TypedDict):
    low: int
    high: int


class LimitsUpdate(TypedDict, total=False):
    low: patch.Change[int]
    high: patch.Change[int]


class Cell(TypedDict):
    value: str
    width: int


class CellUpdate(TypedDict, total=False):
    value: patch.Change[str]
    width: patch.Change[int]


class DocUpdate(TypedDict, total=False):
    name: patch.Change[str]
    group: patch.Change[str]
    config: patch.Struct[ConfigUpdate, Config]
    limits: patch.Change[patch.Struct[LimitsUpdate, Limits]]
    tags: patch.ValueListDict[str]
    cells: patch.TypedMapDict[str, Cell, CellUpdate]


VECTORS = json.loads(
    (Path(__file__).parents[2] / "testdata" / "update" / "vectors.json").read_text()
)["cases"]


class TestVectors:
    @pytest.mark.parametrize("case", VECTORS, ids=[case["name"] for case in VECTORS])
    def test_wire_form(self, case: dict[str, Any]) -> None:
        """Should read and write each shared update as Go and TypeScript do."""
        update = patch.Update[DocUpdate].model_validate(case["update"])
        assert json.loads(update.model_dump_json()) == {
            "guards": [],
            **case["update"],
        }
