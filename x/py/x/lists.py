#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any


def normalize[T](*args: T | tuple[T, ...] | list[T] | None) -> list[T]:
    """Flattens the arguments into a single list. None arguments are skipped.

    :param args: Values, lists, or tuples of values to flatten.
    :returns: The flattened list.
    """
    results: list[T] = []
    for arg in args:
        if isinstance(arg, (list, tuple)):
            results.extend(arg)
        elif arg is not None:
            results.append(arg)
    return results


def all_none(*args: Any) -> bool:
    """Returns True if every argument is None."""
    return all(arg is None for arg in args)


def override[T](*args: T | tuple[T, ...] | list[T] | None) -> list[T] | None:
    for arg in args:
        if arg is not None:
            return normalize(arg)
    return None
