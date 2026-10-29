#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from x.exceptions import Error, ExceptionPayload, Provider, Registry

_TYPE = "test.error"


class _Error(Exception):
    pass


def _encode(exc: Exception) -> ExceptionPayload | None:
    if not isinstance(exc, _Error):
        return None
    return ExceptionPayload(type=_TYPE, data=str(exc))


def _decode(pld: ExceptionPayload) -> Exception | None:
    return _Error(pld.data) if pld.type == _TYPE else None


@pytest.fixture
def registry() -> Registry:
    registry = Registry()
    registry.register(Provider(encode=_encode, decode=_decode))
    return registry


class TestEncode:
    def test_none(self, registry: Registry) -> None:
        """Should encode None as a nil payload."""
        assert registry.encode(None) == ExceptionPayload(type="nil", data="")

    def test_registered(self, registry: Registry) -> None:
        """Should encode an exception through its provider."""
        pld = registry.encode(_Error("cat"))
        assert pld == ExceptionPayload(type=_TYPE, data="cat")

    def test_unknown(self, registry: Registry) -> None:
        """Should encode an unhandled exception as an unknown payload."""
        pld = registry.encode(ValueError("dog"))
        assert pld == ExceptionPayload(type="unknown", data="dog")


class TestDecode:
    def test_none(self, registry: Registry) -> None:
        """Should decode a missing payload as no exception."""
        assert registry.decode(None) is None

    def test_nil(self, registry: Registry) -> None:
        """Should decode a nil payload as no exception."""
        assert registry.decode(ExceptionPayload(type="nil", data="")) is None

    def test_registered(self, registry: Registry) -> None:
        """Should decode a payload through its provider."""
        exc = registry.decode(ExceptionPayload(type=_TYPE, data="cat"))
        assert isinstance(exc, _Error)
        assert str(exc) == "cat"

    def test_unknown(self, registry: Registry) -> None:
        """Should decode an unhandled type into a generic Error."""
        exc = registry.decode(ExceptionPayload(type="other", data="dog"))
        assert type(exc) is Error
        assert str(exc) == "dog"

    def test_round_trip(self, registry: Registry) -> None:
        """Should decode what it encodes."""
        exc = registry.decode(registry.encode(_Error("cat")))
        assert isinstance(exc, _Error)
        assert str(exc) == "cat"

    def test_isolated(self, registry: Registry) -> None:
        """Should not decode through the providers of another registry."""
        pld = ExceptionPayload(type=_TYPE, data="cat")
        assert isinstance(registry.decode(pld), _Error)
        assert type(Registry().decode(pld)) is Error
