#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import freighter
from x.exceptions import Error, ExceptionPayload, Registry


@pytest.fixture
def registry() -> Registry:
    registry = Registry()
    registry.register(freighter.EXCEPTION_PROVIDER)
    return registry


class TestDecode:
    def test_eof(self, registry: Registry) -> None:
        """Should decode a freighter EOF payload."""
        pld = ExceptionPayload(type=freighter.EOF.TYPE, data="EOF")
        assert isinstance(registry.decode(pld), freighter.EOF)

    def test_unknown_type(self, registry: Registry) -> None:
        """Should decode an unknown freighter type into a generic Error."""
        pld = ExceptionPayload(type="freighter.unknown", data="unknown error")
        exc = registry.decode(pld)
        assert type(exc) is Error
        assert str(exc) == "unknown error"


class TestEncode:
    def test_unreachable(self, registry: Registry) -> None:
        """Should encode Unreachable with its message."""
        pld = registry.encode(freighter.Unreachable(message="gone"))
        assert pld == ExceptionPayload(type=freighter.Unreachable.TYPE, data="gone")
