#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Mapping, Sequence
from typing import Any

from pydantic import BaseModel

from synnax.exceptions import MultipleFoundError, NotFoundError, ValidationError


def sole[T](name: str, query: object, results: Sequence[T]) -> T:
    """Returns the only record that a single retrieve found.

    :param name: The kind of record, such as "Range", for the error message.
    :param query: What the caller asked for, for the error message.
    :param results: The records that matched the query.
    :raises NotFoundError: If no record matched.
    :raises MultipleFoundError: If more than one record matched.
    """
    found = sole_or_none(name, query, results)
    if found is None:
        raise NotFoundError(f"{name} matching {query} not found")
    return found


def sole_or_none[T](name: str, query: object, results: Sequence[T]) -> T | None:
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


def request_where[W: BaseModel](
    model: type[W], tree: W | None, fields: Mapping[str, Any]
) -> W | None:
    """Joins tree with the filter fields of a retrieve. Each field names a line of model
    and holds a value, a list, or an operator, as in a partial record.

    :param model: The Where type of the entity.
    :param tree: The tree the caller passed, or None. A tree with no lines adds nothing.
    :param fields: The filter fields, keyed by their Python names.
    :returns: The joined tree, or None when there is neither a tree nor a field.
    :raises ValidationError: If a field is None, since dropping it would select every
        record.
    :raises pydantic.ValidationError: If a field is not a line of model.
    """
    absent = [name for name, value in fields.items() if value is None]
    if absent:
        raise ValidationError(
            f"{', '.join(absent)} is None. Leave a field out to match every record."
        )
    if not fields:
        return tree
    read = model.model_validate(fields)
    if tree is None or tree == model():
        return read
    return model.model_validate({"and": [tree, read]})


def returns_one(
    fields: Mapping[str, Any], one: bool, limit: int | None, offset: int
) -> bool:
    """Reports whether a retrieve returns one record: one is set, or the key or name
    field holds a plain value.

    :raises ValidationError: If such a retrieve sets limit or offset, since a page could
        hide a second match.
    """
    single = one or any(
        f in fields and not isinstance(fields[f], (list, tuple, dict, BaseModel))
        for f in ("key", "name")
    )
    if single and (limit is not None or offset != 0):
        raise ValidationError(
            "A retrieve of one record cannot set limit or offset, since a page could "
            "hide a second match."
        )
    return single
