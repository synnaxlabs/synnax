#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import functools
from collections.abc import Callable
from typing import ParamSpec, TypeVar, overload

P = ParamSpec("P")
R = TypeVar("R")


class RequiresNamedParams(TypeError):
    """
    Custom exception raised when a function is called with positional arguments
    instead of named arguments.
    """

    pass


@overload
def require_named_params(func: Callable[P, R]) -> Callable[P, R]: ...


@overload
def require_named_params(
    func: None = None, *, example_params: tuple[str, str] | None = None
) -> Callable[[Callable[P, R]], Callable[P, R]]: ...


def require_named_params(
    func: Callable[P, R] | None = None,
    *,
    example_params: tuple[str, str] | None = None,
) -> Callable[P, R] | Callable[[Callable[P, R]], Callable[P, R]]:
    """
    Decorator that catches TypeError exceptions related to positional arguments
    and re-raises them with a more helpful error message.

    Args:
        func: The function to decorate
        example_params: Optional tuple of (param_name, param_value) to show in the
            error message. Example: example_params=("user_id", "12345")

    Returns:
        The decorated function with improved error messages for positional arguments
    """

    def decorator(f: Callable[P, R]) -> Callable[P, R]:
        @functools.wraps(f)
        def wrapper(*args: P.args, **kwargs: P.kwargs) -> R:
            try:
                return f(*args, **kwargs)
            except TypeError as e:
                if "positional argument" in str(e) and "were given" in str(e):
                    func_name = f.__qualname__
                    if example_params:
                        param_name, param_value = example_params
                        param_example = f"{func_name}({param_name}='{param_value}')"
                        value_example = f"'{param_value}'"
                    else:
                        param_example = f"{func_name}(name='value')"
                        value_example = "'value'"
                    message = (
                        f"{str(e)}. '{func_name}' only accepts named"
                        f" parameters.\nTry using named parameters"
                        f" like: {param_example} instead of"
                        f" {func_name}({value_example})"
                    )
                    raise RequiresNamedParams(message) from None
                raise

        return wrapper

    if func is None:
        return decorator
    return decorator(func)
