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


class _RequiresNamedParams(Error, TypeError):
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
                    raise _RequiresNamedParams(message) from None
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
    """Lets callers pass the given parameters by position, in the given order, after
    the leading positional parameters that are not among them. A call is remapped and
    emits a DeprecationWarning when it passes more positional arguments than the
    function takes, or passes a positional name by keyword while an argument fills its
    slot. A None remapped into ``**kwargs`` is left out.

    :param names: The parameters that callers could once pass by position. A
        positional name must follow a keyword-only name.
    :raises ValueError: If a name is not a parameter and the function takes no
        ``**kwargs``, or if a positional name has no keyword-only name before it.
    """

    def decorator(f: Callable[P, R]) -> Callable[P, R]:
        params = inspect.signature(f).parameters
        positional = [k for k, p in params.items() if p.kind in _POSITIONAL]
        variadic = any(p.kind == inspect.Parameter.VAR_KEYWORD for p in params.values())
        for i, name in enumerate(names):
            p = params.get(name)
            if p is None and variadic:
                continue
            if p is not None and (
                p.kind == inspect.Parameter.KEYWORD_ONLY
                or (
                    p.kind == inspect.Parameter.POSITIONAL_OR_KEYWORD
                    and any(m not in positional for m in names[:i])
                )
            ):
                continue
            raise ValueError(f"{name} is not a keyword-only parameter of {f}")
        n = len(positional)
        start = min((positional.index(m) for m in names if m in positional), default=n)
        reordered = [m for m in names if m in positional]

        @functools.wraps(f)
        def wrapper(*args: Any, **kwargs: Any) -> R:
            old = len(args) > n or (
                len(args) > start and any(m in kwargs for m in reordered)
            )
            if not old or len(args) - start > len(names):
                return f(*args, **kwargs)
            bound = dict(zip(names, args[start:], strict=False))
            warnings.warn(
                f"Passing {', '.join(bound)} to {f.__qualname__}() by position is "
                "deprecated and stops working in the next minor release. Pass "
                f"{', '.join(bound)} by keyword.",
                DeprecationWarning,
                stacklevel=2,
            )
            moved = {k: v for k, v in bound.items() if v is not None or k in params}
            return f(*args[:start], **kwargs, **moved)

        if inspect.iscoroutinefunction(f):
            inspect.markcoroutinefunction(wrapper)
        return wrapper

    return decorator
