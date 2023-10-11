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


def normalize(*args: T | tuple[T] | list[T]) -> list[T]:
    """Flatten a list of lists into a single list.

    Args:
        *args: A list of lists to flatten.

    Returns:
        A flattened list.
    """
    results: list[T] = list()
    for arg in args:
        if isinstance(arg, (list, tuple)):
            results.extend(arg)
        else:
            results.append(arg)
    return results
