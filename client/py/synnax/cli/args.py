#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TypeVar, Any, Callable

V = TypeVar("V")


def if_none(
    value: V | None,
    func: Callable[[Any], V],
    *args: Any,
    **kwargs: Any,
) -> V:
    """If value is None, return the result of calling func with args."""
    return func(*args, **kwargs) if value is None else value
