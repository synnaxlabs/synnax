#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import BaseModel, ConfigDict, Field, ValidationError

from x import filter
from x.filter import shorthand


class Where(BaseModel):
    model_config = ConfigDict(populate_by_name=True, extra="forbid")

    and_: list["Where"] | None = Field(default=None, alias="and")
    key: filter.Set[str] | None = None


class TestMerge:
    def test_no_where(self) -> None:
        flat = Where(key=filter.Set[str](in_=["a"]))
        assert filter.merge(None, flat) == flat

    def test_empty_flat(self) -> None:
        where = Where(key=filter.Set[str](eq="a"))
        assert filter.merge(where, Where()) == where

    def test_both(self) -> None:
        where = Where(key=filter.Set[str](eq="a"))
        flat = Where(key=filter.Set[str](in_=["b"]))
        assert filter.merge(where, flat) == Where(and_=[where, flat])


class TestShorthand:
    def test_string_value(self) -> None:
        assert filter.String.model_validate("a") == filter.String(eq="a")

    def test_string_list(self) -> None:
        assert filter.String.model_validate(["a", "b"]) == filter.String(in_=["a", "b"])

    def test_string_dict(self) -> None:
        got = filter.String.model_validate({"in": ["a"], "case_insensitive": True})
        assert got == filter.String(in_=["a"], case_insensitive=True)

    def test_operator(self) -> None:
        op = filter.String(starts_with="a")
        assert filter.String.model_validate(op) == op

    def test_set_value_and_tuple(self) -> None:
        assert filter.Set[int].model_validate(1) == filter.Set[int](eq=1)
        assert filter.Set[int].model_validate((1, 2)) == filter.Set[int](in_=[1, 2])

    def test_ordered_dict(self) -> None:
        got = filter.Ordered[int].model_validate({"gt": 1, "lte": 3})
        assert got == filter.Ordered[int](gt=1, lte=3)

    def test_bool(self) -> None:
        assert filter.Bool.model_validate(False) == filter.Bool(eq=False)

    def test_contains(self) -> None:
        assert filter.Contains[str].model_validate("a") == filter.Contains[str](
            any=["a"]
        )
        assert filter.Contains[str].model_validate(["a", "b"]) == filter.Contains[str](
            any=["a", "b"]
        )

    def test_unknown_key(self) -> None:
        with pytest.raises(ValidationError, match="startswith"):
            filter.String.model_validate({"startswith": "a"})

    def test_absent_line(self) -> None:
        assert Where.model_validate({"key": None}).key is None

    def test_line(self) -> None:
        got = Where.model_validate({"and": [{"key": ["a", "b"]}, {"key": "c"}]})
        assert got == Where(
            and_=[
                Where(key=filter.Set[str](in_=["a", "b"])),
                Where(key=filter.Set[str](eq="c")),
            ]
        )

    def test_list_line(self) -> None:
        assert shorthand.read_list("a") == ["a"]
        assert shorthand.read_list(["a"]) == ["a"]
        assert shorthand.read_list(None) is None
