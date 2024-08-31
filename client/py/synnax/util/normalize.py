#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TypeVar

T = TypeVar("T")


def normalize(*args: T | tuple[T] | list[T] | None) -> list[T]:
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
        else:
            results.append(arg)
    return results


def check_for_none(*args: T | None) -> bool:
    """Check if any of the arguments are None.

    Args:
        *args: A list of arguments to check for None.

    Returns:
        True if any of the arguments are None, False otherwise.
    """
    return all(arg is None for arg in args)


def override(
    *args: T | tuple[T] | list[T] | None,
) -> list[T] | None:
    for arg in args:
        if arg is not None:
            return normalize(arg)
    return None
