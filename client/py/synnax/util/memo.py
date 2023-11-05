from functools import wraps


def memo(param: str):
    def decorator(func):
        @wraps(func)
        def wrapper(self, *args, **kwargs):
            if not hasattr(self, param):
                setattr(self, param, func(self, *args, **kwargs))
            return getattr(self, param)

        return wrapper

    return decorator
