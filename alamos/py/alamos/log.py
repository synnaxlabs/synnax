#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from collections.abc import Mapping
from logging import Logger as BaseLogger
from typing import Any

from alamos.meta import InstrumentationMeta
from alamos.noop import Noop
from alamos.noop import noop as noopd


class Logger:
    """
    Logger wraps Python's logging implementation to provide an opinionated formatter and
    no-op logging functionality. A child logger adds the key and path of its
    instrumentation to the extra fields of each record, so a formatter can show them
    with %(key)s and %(path)s.
    """

    noop: bool = True
    base: BaseLogger | None = None
    meta: InstrumentationMeta
    _fields: dict[str, str]

    def _(self) -> Noop:
        return self

    def __init__(self, *, noop: bool = True, base: BaseLogger | None = None):
        self.noop = noop
        self.base = base
        self._fields = {}

    def _extra(self, extra: Mapping[str, object] | None) -> dict[str, object]:
        return {**(extra or {}), **self._fields}

    @noopd
    def debug(self, /, msg: str, *args: Any, **kwargs: Any) -> None:
        """Logs a message at the Debug level"""
        if self.base is not None:
            extra = self._extra(kwargs.pop("extra", None))
            self.base.debug(msg, *args, extra=extra, **kwargs)

    @noopd
    def info(self, /, msg: str, *args: Any, **kwargs: Any) -> None:
        """Logs a message at the Info level"""
        if self.base is not None:
            extra = self._extra(kwargs.pop("extra", None))
            self.base.info(msg, *args, extra=extra, **kwargs)

    @noopd
    def warn(self, /, msg: str, *args: Any, **kwargs: Any) -> None:
        """Logs a message at the Warn level"""
        if self.base is not None:
            extra = self._extra(kwargs.pop("extra", None))
            self.base.warning(msg, *args, extra=extra, **kwargs)

    @noopd
    def error(self, /, msg: str, *args: Any, **kwargs: Any) -> None:
        """Logs a message at the Error level"""
        if self.base is not None:
            extra = self._extra(kwargs.pop("extra", None))
            self.base.error(msg, *args, extra=extra, **kwargs)

    def child_(self, /, meta: InstrumentationMeta) -> Logger:
        logger = Logger(noop=self.noop, base=self.base)
        logger.meta = meta
        logger._fields = {"key": meta.key, "path": meta.path}
        return logger


NOOP_LOGGER = Logger()
