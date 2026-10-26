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
