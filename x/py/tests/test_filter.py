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


class Span(BaseModel):
    start: int
    end: int


class ThingWhere(BaseModel):
    model_config = ConfigDict(populate_by_name=True, extra="forbid")

    and_: list["ThingWhere"] | None = Field(default=None, alias="and")
    or_: list["ThingWhere"] | None = Field(default=None, alias="or")
    not_: "ThingWhere | None" = Field(default=None, alias="not")
    key: filter.Set[int] | None = None
    name: filter.String | None = None
    count: filter.Ordered[int] | None = None
    tags: filter.Contains[str] | None = None
    virtual: bool | None = None
    labels: list[str] | None = None
    active: filter.Bool | None = None
    span: Span | None = None


def _name(op: filter.String) -> ThingWhere:
    return ThingWhere(name=op)


class TestDescribe:
    def test_operators(self) -> None:
        """Should describe each operator."""
        assert filter.describe(_name(filter.String(eq="press"))) == 'name = "press"'
        where = ThingWhere(key=filter.Set[int](in_=[12, 14]))
        assert filter.describe(where) == "key in [12, 14]"
        assert (
            filter.describe(_name(filter.String(contains="a"))) == 'name contains "a"'
        )
        where = _name(filter.String(starts_with="a"))
        assert filter.describe(where) == 'name starts with "a"'
        where = _name(filter.String(regex="pre.*"))
        assert filter.describe(where) == "name matches /pre.*/"
        where = ThingWhere(count=filter.Ordered[int](gt=3))
        assert filter.describe(where) == "count > 3"
        where = ThingWhere(count=filter.Ordered[int](lte=3))
        assert filter.describe(where) == "count <= 3"
        where = ThingWhere(count=filter.Ordered[int](gte=3, lt=10))
        assert filter.describe(where) == "count >= 3 and count < 10"
        where = ThingWhere(tags=filter.Contains[str](any=["a", "b"]))
        assert filter.describe(where) == 'tags has any of ["a", "b"]'
        where = ThingWhere(active=filter.Bool(eq=True))
        assert filter.describe(where) == "active = true"

    def test_case_insensitive(self) -> None:
        """Should mark each operator of a case-insensitive condition."""
        where = _name(filter.String(eq="Press", case_insensitive=True))
        assert filter.describe(where) == 'name = "Press" (ignoring case)'
        where = _name(
            filter.String(contains="a", starts_with="b", case_insensitive=True)
        )
        assert filter.describe(where) == (
            'name contains "a" (ignoring case) and name starts with "b" (ignoring case)'
        )

    def test_value(self) -> None:
        """Should describe a value that is not an operator object."""
        assert filter.describe(ThingWhere(virtual=True)) == "virtual = true"
        assert filter.describe(ThingWhere(labels=["a", "b"])) == 'labels = ["a", "b"]'
        where = ThingWhere(span=Span(start=1, end=2))
        assert filter.describe(where) == 'span = {"end":2,"start":1}'

    def test_escape(self) -> None:
        """Should escape a string."""
        assert filter.describe(_name(filter.String(eq='a"b'))) == 'name = "a\\"b"'

    def test_sort(self) -> None:
        """Should sort the fields of a node by name."""
        where = ThingWhere(
            name=filter.String(eq="a"),
            active=filter.Bool(eq=True),
            key=filter.Set[int](eq=1),
        )
        assert filter.describe(where) == 'active = true and key = 1 and name = "a"'

    def test_and(self) -> None:
        """Should join the children of and."""
        where = ThingWhere(
            and_=[_name(filter.String(eq="a")), ThingWhere(key=filter.Set[int](eq=1))]
        )
        assert filter.describe(where) == 'name = "a" and key = 1'

    def test_or(self) -> None:
        """Should join the children of or."""
        where = ThingWhere(
            or_=[
                _name(filter.String(eq="a")),
                ThingWhere(not_=ThingWhere(active=filter.Bool(eq=True))),
            ]
        )
        assert filter.describe(where) == 'name = "a" or not (active = true)'

    def test_or_beside_terms(self) -> None:
        """Should wrap an or in parentheses beside other terms."""
        where = ThingWhere(
            key=filter.Set[int](eq=1),
            or_=[_name(filter.String(eq="a")), _name(filter.String(eq="b"))],
        )
        assert filter.describe(where) == 'key = 1 and (name = "a" or name = "b")'

    def test_single_or_child_grouping(self) -> None:
        """Should keep the grouping of a child of a single-child or."""
        inner = ThingWhere(
            or_=[_name(filter.String(eq="a")), _name(filter.String(eq="b"))]
        )
        where = ThingWhere(key=filter.Set[int](eq=1), or_=[inner])
        assert filter.describe(where) == 'key = 1 and (name = "a" or name = "b")'

    def test_or_child_terms(self) -> None:
        """Should wrap a child of or that has more than one term."""
        where = ThingWhere(
            or_=[
                ThingWhere(key=filter.Set[int](eq=1), name=filter.String(eq="a")),
                ThingWhere(key=filter.Set[int](eq=2)),
            ]
        )
        assert filter.describe(where) == '(key = 1 and name = "a") or key = 2'

    def test_not(self) -> None:
        """Should describe not."""
        where = ThingWhere(
            not_=ThingWhere(key=filter.Set[int](eq=1), name=filter.String(eq="a"))
        )
        assert filter.describe(where) == 'not (key = 1 and name = "a")'

    def test_empty(self) -> None:
        """Should describe an empty tree as an empty string."""
        assert filter.describe(None) == ""
        assert filter.describe(ThingWhere()) == ""
        assert filter.describe(ThingWhere(and_=[])) == ""
        assert filter.describe(_name(filter.String())) == ""

    def test_nested_empty(self) -> None:
        """Should describe nested empty trees."""
        one = ThingWhere(key=filter.Set[int](eq=1))
        assert filter.describe(ThingWhere(or_=[ThingWhere(), one])) == "true or key = 1"
        where = ThingWhere(or_=[ThingWhere(and_=[]), one])
        assert filter.describe(where) == "true or key = 1"
        assert filter.describe(ThingWhere(or_=[])) == "false"
        assert filter.describe(ThingWhere(or_=[ThingWhere()])) == ""
        assert filter.describe(ThingWhere(not_=ThingWhere())) == "not (true)"
