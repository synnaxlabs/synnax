from alamos.tracer import Tracer, trace

from alamos.logger import Logger


class Instrumentation:
    l: Logger
    t: Tracer

    def __init__(self, l: Logger, t: Tracer):
        self.l = l
        self.t = t
