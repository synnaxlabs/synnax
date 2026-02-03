#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable
from typing import Concatenate, ParamSpec, Protocol, TypeVar


class Noop(Protocol):
    """A protocol for a class that can be marked as noop based on a boolean flag."""

    noop: bool


P = ParamSpec("P")
T = TypeVar("T", bound=Noop)


def noop(
    f: Callable[Concatenate[T, P], None],
) -> Callable[Concatenate[T, P], None]:
    """
    Decorator for methods on a Noop class. If the instance's `noop` flag is True, the
    decorated function call is skipped and returns None. Otherwise, it executes normally.
    """

    def wrapper(self: T, /, *args: P.args, **kwargs: P.kwargs) -> None:
        if self.noop:
            return
        return f(self, *args, **kwargs)

    return wrapper
