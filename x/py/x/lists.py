#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any


def none_to_empty(v: Any) -> Any:
    """Coerce None to an empty list for required array fields.

    The wire treats a null array the same as an absent one: no items. Used as a
    pydantic BeforeValidator so an explicit null validates as [] instead of raising.
    """
    return [] if v is None else v


def normalize[T](*args: T | tuple[T] | list[T] | None) -> list[T]:
    """Flatten a list of lists into a single list.

    Args:
        *args: A list of lists to flatten.

    Returns:
        A flattened list.
    """
    results: list[T] = list()
    if args[0] is None:
        return results
    for arg in args:
        if isinstance(arg, (list, tuple)):
            results.extend(arg)
        elif arg is not None:
            results.append(arg)
    return results


def check_for_none(*args: Any) -> bool:
    """Check if any of the arguments are None.

    Args:
        *args: A list of arguments to check for None.

    Returns:
        True if any of the arguments are None, False otherwise.
    """
    return all(arg is None for arg in args)


def override[T](*args: T | tuple[T] | list[T] | None) -> list[T] | None:
    for arg in args:
        if arg is not None:
            return normalize(arg)
    return None
