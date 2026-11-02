#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import logging

import pytest

from alamos import Instrumentation, Logger
from alamos.log import NOOP_LOGGER


class TestLog:
    def test_noop(self, caplog: pytest.LogCaptureFixture) -> None:
        """Should not emit any records."""
        with caplog.at_level(logging.DEBUG):
            NOOP_LOGGER.debug("test")
            NOOP_LOGGER.info("test")
            NOOP_LOGGER.warn("test")
            NOOP_LOGGER.error("test")
        assert caplog.records == []

    def test_noop_with_base(
        self, base: logging.Logger, caplog: pytest.LogCaptureFixture
    ) -> None:
        """Should not emit records when noop, even with a base logger."""
        with caplog.at_level(logging.DEBUG):
            Logger(noop=True, base=base).info("test")
        assert caplog.records == []

    def test_levels(
        self, base: logging.Logger, caplog: pytest.LogCaptureFixture
    ) -> None:
        """Should emit each message at the matching level."""
        logger = Logger(noop=False, base=base)
        with caplog.at_level(logging.DEBUG):
            logger.debug("debug")
            logger.info("info")
            logger.warn("warn")
            logger.error("error")
        assert [(r.levelno, r.getMessage()) for r in caplog.records] == [
            (logging.DEBUG, "debug"),
            (logging.INFO, "info"),
            (logging.WARNING, "warn"),
            (logging.ERROR, "error"),
        ]

    def test_none_extra(
        self, base: logging.Logger, caplog: pytest.LogCaptureFixture
    ) -> None:
        """Should accept None as the extra fields at every level."""
        logger = Logger(noop=False, base=base)
        with caplog.at_level(logging.DEBUG):
            logger.debug("debug", extra=None)
            logger.info("info", extra=None)
            logger.warn("warn", extra=None)
            logger.error("error", extra=None)
        assert len(caplog.records) == 4

    def test_args(self, base: logging.Logger, caplog: pytest.LogCaptureFixture) -> None:
        """Should format the message with the given arguments."""
        with caplog.at_level(logging.INFO):
            Logger(noop=False, base=base).info("value %d", 1)
        (record,) = caplog.records
        assert record.getMessage() == "value 1"


class Records(logging.Handler):
    def __init__(self) -> None:
        super().__init__()
        self.records: list[logging.LogRecord] = []

    def emit(self, record: logging.LogRecord) -> None:
        self.records.append(record)


class TestInstrumentationLogger:
    def test_meta_fields(
        self, base: logging.Logger, caplog: pytest.LogCaptureFixture
    ) -> None:
        """Should add the instrumentation key and path to each record."""
        ins = Instrumentation(key="root", logger=Logger(noop=False, base=base))
        with caplog.at_level(logging.INFO):
            ins.L.info("root")
            ins.child("a").child("b").L.info("child", extra={"user": "x"})
        root, child = caplog.records
        assert (root.__dict__["key"], root.__dict__["path"]) == ("root", "root")
        assert (child.__dict__["key"], child.__dict__["path"]) == ("b", "root.a.b")
        assert child.__dict__["user"] == "x"

    def test_supplied_base(self) -> None:
        """Should send records to the handlers of the supplied base logger."""
        handler = Records()
        base = logging.Logger("app")
        base.addHandler(handler)
        ins = Instrumentation(key="root", logger=Logger(noop=False, base=base))
        ins.child("a").L.warn("child")
        (record,) = handler.records
        assert record.name == "app"
        assert record.__dict__["path"] == "root.a"

    def test_meta(self, base: logging.Logger) -> None:
        """Should set the instrumentation meta on the logger."""
        child = Instrumentation(key="root", logger=Logger(noop=False, base=base)).child(
            "a"
        )
        assert child.L.meta.key == "a"
        assert child.L.meta.path == "root.a"
