#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from logging import Logger as BaseLogger

from alamos.meta import InstrumentationMeta
from alamos.noop import noop as noopd, Noop


class Logger:
    """Logger wraps Python's logging implementation to provide an opinionated formatter
    and no-op logging functionality.
    """

    noop: bool = True
    base: BaseLogger
    meta: InstrumentationMeta

    def _(self) -> Noop:
        return self

    def __init__(self, noop: bool = True, base: BaseLogger = None):
        self.noop = noop
        self.base = base

    @noopd
    def debug(self, msg: str, *args, **kwargs):
        """Logs a message at the Debug level"""
        self.base.debug(msg, *args, **kwargs)

    @noopd
    def info(self, msg: str, *args, **kwargs):
        """Logs a message at the Info level"""
        self.base.info(msg, *args, **kwargs)

    @noopd
    def warn(self, msg: str, *args, **kwargs):
        """Logs a message at the Warn level"""
        self.base.warning(msg, *args, **kwargs)

    @noopd
    def error(self, msg: str, *args, **kwargs):
        """Logs a message at the Error level"""
        self.base.error(msg, *args, **kwargs)

    def child_(self, meta: InstrumentationMeta) -> Logger:
        l = Logger(noop=self.noop, base=self.base)
        l.meta = meta
        return l


NOOP_LOGGER = Logger()
