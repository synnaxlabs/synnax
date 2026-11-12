#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


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
