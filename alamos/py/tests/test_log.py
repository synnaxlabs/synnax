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


@pytest.fixture
def base() -> logging.Logger:
    return logging.getLogger("alamos_test")


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

    def test_args(self, base: logging.Logger, caplog: pytest.LogCaptureFixture) -> None:
        """Should format the message with the given arguments."""
        with caplog.at_level(logging.INFO):
            Logger(noop=False, base=base).info("value %d", 1)
        (record,) = caplog.records
        assert record.getMessage() == "value 1"


class TestInstrumentationLogger:
    def test_path(self, base: logging.Logger, caplog: pytest.LogCaptureFixture) -> None:
        """Should name each record with the instrumentation path."""
        ins = Instrumentation("root", logger=Logger(noop=False, base=base))
        with caplog.at_level(logging.INFO):
            ins.L.info("root")
            ins.child("a").child("b").L.info("child")
        assert [r.name for r in caplog.records] == [
            "alamos_test.root",
            "alamos_test.root.a.b",
        ]

    def test_meta(self, base: logging.Logger) -> None:
        """Should set the instrumentation meta on the logger."""
        child = Instrumentation("root", logger=Logger(noop=False, base=base)).child("a")
        assert child.L.meta.key == "a"
        assert child.L.meta.path == "root.a"
