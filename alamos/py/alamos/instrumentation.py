from alamos.tracer import Tracer, trace

from alamos.logger import Logger


class Instrumentation:
    l: Logger

    def __init__(self, l: Logger, t: Tracer):
        self.l = l
        self.t = t

    @trace("tracethis")
    def tracethis(self, f):
        def wrapper(*args, **kwargs):
            with self.t.trace(f.__name__):
                return f(*args, **kwargs)

        return wrapper
