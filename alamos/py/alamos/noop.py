#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Protocol, ParamSpec, Concatenate, Callable, TypeVar


class Noop(Protocol):
    """A protocol for a class that can be marked as noop based on a boolean"""

    noop: bool


P = ParamSpec("P")
T = TypeVar("T", bound=Noop)


def noop(
    f: Callable[Concatenate[T, P], None],
) -> Callable[Concatenate[T, P], None]:
    """Decorator around a Noop class that will not call the decorated function if the
    Noop.noop is True.
    """

    def wrapper(self: T, *args: P.args, **kwargs: P.kwargs) -> None:
        if self.noop:
            return
        return f(self, *args, **kwargs)

    return wrapper
