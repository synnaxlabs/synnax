#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import functools
import inspect
import warnings
from collections.abc import Callable
from typing import Any, ParamSpec, TypeVar, overload

from x.exceptions import Error

P = ParamSpec("P")
R = TypeVar("R")


class RequiresNamedParams(Error, TypeError):
    """
    Custom exception raised when a function is called with positional arguments
    instead of named arguments.
    """


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

    :param func: The function to decorate
    :param example_params: Optional tuple of (param_name, param_value) to show in the
        error message. Example: example_params=("user_id", "12345")
    :returns: The decorated function with improved error messages for positional
        arguments
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
                        f"{e!s}. '{func_name}' only accepts named"
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


_POSITIONAL = (
    inspect.Parameter.POSITIONAL_ONLY,
    inspect.Parameter.POSITIONAL_OR_KEYWORD,
)


def deprecate_positional(
    *names: str,
) -> Callable[[Callable[P, R]], Callable[P, R]]:
    """Lets callers pass the given keyword-only parameters by position, in the given
    order, after the positional parameters. Each such call emits a DeprecationWarning.

    :param names: The keyword-only parameters that callers could once pass by position.
    :raises ValueError: If a name is not a keyword-only parameter of the function.
    """

    def decorator(f: Callable[P, R]) -> Callable[P, R]:
        params = inspect.signature(f).parameters
        for name in names:
            if (
                name not in params
                or params[name].kind != inspect.Parameter.KEYWORD_ONLY
            ):
                raise ValueError(f"{name} is not a keyword-only parameter of {f}")
        n = sum(1 for p in params.values() if p.kind in _POSITIONAL)

        @functools.wraps(f)
        def wrapper(*args: Any, **kwargs: Any) -> R:
            extra = args[n:]
            if len(extra) == 0 or len(extra) > len(names):
                return f(*args, **kwargs)
            moved = dict(zip(names, extra, strict=False))
            warnings.warn(
                f"Passing {', '.join(moved)} to {f.__qualname__}() by position is "
                "deprecated and stops working in the next minor release. Pass "
                f"{', '.join(moved)} by keyword.",
                DeprecationWarning,
                stacklevel=2,
            )
            return f(*args[:n], **kwargs, **moved)

        if inspect.iscoroutinefunction(f):
            inspect.markcoroutinefunction(wrapper)
        return wrapper

    return decorator
