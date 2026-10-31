#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel, ConfigDict, Field

from x import filter


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


class Thing(filter.Condition):
    pass


class Other(filter.Condition):
    pass


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


key: filter.SetLine[Thing, int] = filter.SetLine(Thing, "key")
name: filter.StringLine[Thing] = filter.StringLine(Thing, "name")
count: filter.OrderedLine[Thing, int] = filter.OrderedLine(Thing, "count")
tags: filter.ContainsLine[Thing, str] = filter.ContainsLine(Thing, "tags")
virtual: filter.FlagLine[Thing] = filter.FlagLine(Thing, "virtual")
labels: filter.ListLine[Thing, str] = filter.ListLine(Thing, "labels")


class TestBuilder:
    def test_set(self) -> None:
        """Should build eq and in_ on a set line."""
        assert key.eq(1).tree == {"key": {"eq": 1}}
        assert key.in_([1, 2]).tree == {"key": {"in_": [1, 2]}}

    def test_ordered(self) -> None:
        """Should build comparisons on an ordered line."""
        assert count.gt(3).tree == {"count": {"gt": 3}}
        assert count.lte(3).tree == {"count": {"lte": 3}}

    def test_string(self) -> None:
        """Should build text operators with the case flag."""
        assert name.contains("a").tree == {"name": {"contains": "a"}}
        assert name.starts_with("a", case_insensitive=True).tree == {
            "name": {"starts_with": "a", "case_insensitive": True}
        }

    def test_contains(self) -> None:
        """Should build has and has_any on a contains line."""
        assert tags.has("a").tree == {"tags": {"any": ["a"]}}
        assert tags.has_any(["a", "b"]).tree == {"tags": {"any": ["a", "b"]}}

    def test_typed_lines(self) -> None:
        """Should build the bare value of each typed line."""
        assert virtual.eq(False).tree == {"virtual": False}
        assert labels.has("a").tree == {"labels": ["a"]}
        window: filter.ValueLine[Thing, int] = filter.ValueLine(Thing, "window")
        assert window(3).tree == {"window": 3}

    def test_operators(self) -> None:
        """Should join conditions with &, |, and ~."""
        assert (key.eq(1) & name.eq("a")).tree == {
            "and_": [{"key": {"eq": 1}}, {"name": {"eq": "a"}}]
        }
        assert (key.eq(1) | key.eq(2)).tree == {
            "or_": [{"key": {"eq": 1}}, {"key": {"eq": 2}}]
        }
        assert (~key.eq(1)).tree == {"not_": {"key": {"eq": 1}}}

    def test_repr(self) -> None:
        """Should show the entity and the tree."""
        assert repr(key.eq(1)) == "Thing({'key': {'eq': 1}})"

    def test_tree_functions(self) -> None:
        """Should join conditions with and_, or_, and not_."""
        assert Thing.and_(key.eq(1), key.eq(2), key.eq(3)).tree == {
            "and_": [{"key": {"eq": 1}}, {"key": {"eq": 2}}, {"key": {"eq": 3}}]
        }
        assert Thing.or_(key.eq(1)).tree == {"or_": [{"key": {"eq": 1}}]}
        assert Thing.not_(key.eq(1)).tree == {"not_": {"key": {"eq": 1}}}

    def test_type_errors(self) -> None:
        """Should fail the type check for wrong entities and values."""
        other: filter.SetLine[Other, int] = filter.SetLine(Other, "key")
        _ = key.eq(1) & other.eq(1)  # type: ignore[operator]
        _ = Thing.and_(key.eq(1), other.eq(1))  # type: ignore[arg-type]
        _ = key.eq("1")  # type: ignore[arg-type]
        _ = key.eq(None)  # type: ignore[arg-type]


class TestJoin:
    def test_no_conditions(self) -> None:
        """Should join no conditions into no tree."""
        assert filter.join(ThingWhere, []) is None

    def test_one_condition(self) -> None:
        """Should join one condition into its own tree."""
        assert filter.join(ThingWhere, [key.eq(1)]) == ThingWhere(
            key=filter.Set[int](eq=1)
        )

    def test_conditions(self) -> None:
        """Should join several conditions under and_."""
        joined = filter.join(
            ThingWhere, [key.in_([1]), name.eq("a", case_insensitive=True)]
        )
        assert joined == ThingWhere(
            and_=[
                ThingWhere(key=filter.Set[int](in_=[1])),
                ThingWhere(name=filter.String(eq="a", case_insensitive=True)),
            ]
        )

    def test_stored_tree(self) -> None:
        """Should keep a stored tree as one condition."""
        tree = ThingWhere(virtual=True)
        assert filter.join(ThingWhere, [Thing(tree), labels.has("a")]) == ThingWhere(
            and_=[tree, ThingWhere(labels=["a"])]
        )


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
