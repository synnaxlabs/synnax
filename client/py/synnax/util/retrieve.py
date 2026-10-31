#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Sequence

from synnax.exceptions import MultipleFoundError, NotFoundError
from x import filter


def one[T](name: str, query: object, results: Sequence[T]) -> T:
    """Returns the only record that a single retrieve found.

    :param name: The kind of record, such as "Range", for the error message.
    :param query: What the caller asked for, for the error message.
    :param results: The records that matched the query.
    :raises NotFoundError: If no record matched.
    :raises MultipleFoundError: If more than one record matched.
    """
    found = one_or_none(name, query, results)
    if found is None:
        raise NotFoundError(f"{name} matching {query} not found")
    return found


def one_or_none[T](name: str, query: object, results: Sequence[T]) -> T | None:
    """Returns the only record that a single retrieve found, or None if no record
    matched.

    :param name: The kind of record, such as "Range", for the error message.
    :param query: What the caller asked for, for the error message.
    :param results: The records that matched the query.
    :raises MultipleFoundError: If more than one record matched.
    """
    if len(results) > 1:
        raise MultipleFoundError(
            f"Expected one {name} matching {query}, but found {len(results)}"
        )
    return results[0] if results else None


def conditions[C: filter.Condition](
    kind: str, args: Sequence[object], condition: type[C]
) -> list[C]:
    """Returns args, the conditions of a retrieve.

    :param kind: The kind of record, such as "Range", for the error message.
    :param args: The positional arguments of the retrieve.
    :param condition: The condition class of the entity.
    :raises TypeError: If an argument is not a condition of the entity.
    """
    conds: list[C] = []
    for arg in args:
        if not isinstance(arg, condition):
            raise TypeError(f"{arg!r} is not a {kind} condition")
        conds.append(arg)
    return conds


def _names[C: filter.Condition](
    line: filter.StringLine[C] | filter.ListLine[C, str], names: list[str]
) -> C:
    return line.has_any(names) if isinstance(line, filter.ListLine) else line.in_(names)


def shortcut[C: filter.Condition, K](
    kind: str,
    args: Sequence[object],
    condition: type[C],
    key_type: type[K],
    key: filter.SetLine[C, K],
    name: filter.StringLine[C] | filter.ListLine[C, str] | None = None,
) -> tuple[list[C], object]:
    """Returns the conditions that the positional arguments of a retrieve make, and the
    key or name that asks for one record, or None.

    :param kind: The kind of record, such as "Range", for the error message.
    :param args: Conditions of the entity, or one key, name, or list of either.
    :param condition: The condition class of the entity.
    :param key_type: The type of the entity's keys.
    :param key: The key line of the entity.
    :param name: The name line of the entity, or None if the retrieve takes no names.
    :raises TypeError: If args are neither conditions of the entity nor a shortcut.
    """
    if len(args) != 1 or isinstance(args[0], filter.Condition):
        return conditions(kind, args, condition), None
    value = args[0]
    if isinstance(value, key_type):
        return [key.eq(value)], value
    if name is not None and isinstance(value, str):
        return [_names(name, [value])], value
    if isinstance(value, (list, tuple)):
        keys = [v for v in value if isinstance(v, key_type)]
        if len(keys) == len(value):
            return [key.in_(keys)], None
        names = [v for v in value if isinstance(v, str)]
        if name is not None and len(names) == len(value):
            return [_names(name, names)], None
    raise TypeError(f"{value!r} is not a {kind} condition, key, or name")
