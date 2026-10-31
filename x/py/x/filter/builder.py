#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Sequence
from typing import Self

from pydantic import BaseModel


class Condition:
    """A condition on the records of one entity, which holds a filter tree. Build one
    from the lines of an entity, such as ``ranger.name.eq("Hotfire")``. ``&``, ``|``,
    and ``~`` join and negate the conditions of one entity.
    """

    def __init__(self, tree: object) -> None:
        self.tree = tree

    def __repr__(self) -> str:
        return f"{type(self).__name__}({self.tree!r})"

    @classmethod
    def and_(cls, *conditions: Self) -> Self:
        """Matches the records that every condition matches."""
        return cls({"and_": [c.tree for c in conditions]})

    @classmethod
    def or_(cls, *conditions: Self) -> Self:
        """Matches the records that any condition matches."""
        return cls({"or_": [c.tree for c in conditions]})

    @classmethod
    def not_(cls, condition: Self) -> Self:
        """Matches the records that condition does not match."""
        return cls({"not_": condition.tree})

    def __and__(self, other: Self) -> Self:
        return self.and_(self, other)

    def __or__(self, other: Self) -> Self:
        return self.or_(self, other)

    def __invert__(self) -> Self:
        return self.not_(self)


def _op(line: str, op: str, value: object, case_insensitive: bool = False) -> object:
    ops = {op: value, "case_insensitive": True} if case_insensitive else {op: value}
    return {line: ops}


class SetLine[C: Condition, V]:
    """A line of the operator type Set."""

    def __init__(self, condition: type[C], line: str) -> None:
        self._condition = condition
        self._line = line

    def eq(self, value: V) -> C:
        """Matches a value equal to value."""
        return self._condition(_op(self._line, "eq", value))

    def in_(self, values: Sequence[V]) -> C:
        """Matches a value equal to an element of values."""
        return self._condition(_op(self._line, "in_", list(values)))


class BoolLine[C: Condition]:
    """A line of the operator type Bool."""

    def __init__(self, condition: type[C], line: str) -> None:
        self._condition = condition
        self._line = line

    def eq(self, value: bool) -> C:
        """Matches a value equal to value."""
        return self._condition(_op(self._line, "eq", value))


class OrderedLine[C: Condition, V]:
    """A line of the operator type Ordered."""

    def __init__(self, condition: type[C], line: str) -> None:
        self._condition = condition
        self._line = line

    def eq(self, value: V) -> C:
        """Matches a value equal to value."""
        return self._condition(_op(self._line, "eq", value))

    def in_(self, values: Sequence[V]) -> C:
        """Matches a value equal to an element of values."""
        return self._condition(_op(self._line, "in_", list(values)))

    def gt(self, value: V) -> C:
        """Matches a value greater than value."""
        return self._condition(_op(self._line, "gt", value))

    def gte(self, value: V) -> C:
        """Matches a value greater than or equal to value."""
        return self._condition(_op(self._line, "gte", value))

    def lt(self, value: V) -> C:
        """Matches a value less than value."""
        return self._condition(_op(self._line, "lt", value))

    def lte(self, value: V) -> C:
        """Matches a value less than or equal to value."""
        return self._condition(_op(self._line, "lte", value))


class StringLine[C: Condition]:
    """A line of the operator type String. Each operator ignores case when
    case_insensitive is True.
    """

    def __init__(self, condition: type[C], line: str) -> None:
        self._condition = condition
        self._line = line

    def eq(self, value: str, *, case_insensitive: bool = False) -> C:
        """Matches a value equal to value."""
        return self._condition(_op(self._line, "eq", value, case_insensitive))

    def in_(self, values: Sequence[str], *, case_insensitive: bool = False) -> C:
        """Matches a value equal to an element of values."""
        return self._condition(_op(self._line, "in_", list(values), case_insensitive))

    def contains(self, value: str, *, case_insensitive: bool = False) -> C:
        """Matches a value that holds value."""
        return self._condition(_op(self._line, "contains", value, case_insensitive))

    def starts_with(self, value: str, *, case_insensitive: bool = False) -> C:
        """Matches a value that begins with value."""
        return self._condition(_op(self._line, "starts_with", value, case_insensitive))

    def regex(self, pattern: str, *, case_insensitive: bool = False) -> C:
        """Matches a value that the unanchored RE2 pattern matches."""
        return self._condition(_op(self._line, "regex", pattern, case_insensitive))


class ContainsLine[C: Condition, V]:
    """A line of the operator type Contains."""

    def __init__(self, condition: type[C], line: str) -> None:
        self._condition = condition
        self._line = line

    def has(self, value: V) -> C:
        """Matches an array that holds value."""
        return self._condition(_op(self._line, "any", [value]))

    def has_any(self, values: Sequence[V]) -> C:
        """Matches an array that holds an element of values."""
        return self._condition(_op(self._line, "any", list(values)))


class FlagLine[C: Condition]:
    """A typed line of type bool, whose value goes on the line."""

    def __init__(self, condition: type[C], line: str) -> None:
        self._condition = condition
        self._line = line

    def eq(self, value: bool) -> C:
        """Matches the records that the line holds for value."""
        return self._condition({self._line: value})


class ListLine[C: Condition, V]:
    """A typed line of a list type, whose values go on the line."""

    def __init__(self, condition: type[C], line: str) -> None:
        self._condition = condition
        self._line = line

    def has(self, value: V) -> C:
        """Matches the records that the line holds for value."""
        return self._condition({self._line: [value]})

    def has_any(self, values: Sequence[V]) -> C:
        """Matches the records that the line holds for any element of values."""
        return self._condition({self._line: list(values)})


class ValueLine[C: Condition, V]:
    """A typed line, called with the value that goes on the line."""

    def __init__(self, condition: type[C], line: str) -> None:
        self._condition = condition
        self._line = line

    def __call__(self, value: V) -> C:
        return self._condition({self._line: value})


def join[W: BaseModel](where: type[W], conditions: Sequence[Condition]) -> W | None:
    """Returns the tree that every condition matches.

    :param where: The filter type of the entity.
    :param conditions: The conditions to join.
    :returns: The tree, or None when there are no conditions.
    """
    if len(conditions) == 0:
        return None
    if len(conditions) == 1:
        return where.model_validate(conditions[0].tree)
    return where.model_validate({"and_": [c.tree for c in conditions]})
