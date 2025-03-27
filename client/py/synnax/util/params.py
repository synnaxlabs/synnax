import functools
from typing import Any, Callable, TypeVar, cast

# Define a generic type variable for the function
F = TypeVar("F", bound=Callable[..., Any])


class RequiresNamedParams(TypeError):
    """
    Custom exception raised when a function is called with positional arguments
    instead of named arguments.
    """

    pass


def require_named_params(
    func: F | None = None, *, example_params: tuple[str, str] | None = None
) -> Callable[[F], F]:
    """
    Decorator that catches TypeError exceptions related to positional arguments
    and re-raises them with a more helpful error message.

    Args:
        func: The function to decorate
        example_params: Optional tuple of (param_name, param_value) to show in the error message
                        Example: example_params=("user_id", "12345")

    Returns:
        The decorated function with improved error messages for positional arguments
    """

    def decorator(func: F) -> F:
        @functools.wraps(func)
        def wrapper(*args: Any, **kwargs: Any) -> Any:
            try:
                return func(*args, **kwargs)
            except TypeError as e:
                # Check if this is the "takes X positional arguments but Y were given" error
                if "positional argument" in str(e) and "were given" in str(e):
                    func_name = func.__qualname__

                    # Use custom example if provided, otherwise use generic example
                    if example_params:
                        param_name, param_value = example_params
                        param_example = f"{func_name}({param_name}='{param_value}')"
                        value_example = f"'{param_value}'"
                    else:
                        param_example = f"{func_name}(name='value')"
                        value_example = "'value'"

                    message = (
                        f"{str(e)}. '{func_name}' only accepts named parameters.\n"
                        f"Try using named parameters like: {param_example} instead of {func_name}({value_example})"
                    )
                    raise RequiresNamedParams(message) from None
                # Re-raise other TypeErrors unchanged
                raise

        return cast(F, wrapper)

    # Handle both @require_named_params and @require_named_params(example_params="...")
    if func is None:
        return decorator
    return decorator(func)
