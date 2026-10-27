#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

import pytest

import freighter
import synnax as sy
from synnax.exceptions import create_registry
from x.exceptions import ExceptionPayload


class TestCreateRegistry:
    def test_freighter(self) -> None:
        """Should decode the errors of Freighter."""
        pld = ExceptionPayload(type=freighter.EOF.TYPE, data="EOF")
        assert isinstance(create_registry().decode(pld), freighter.EOF)

    def test_path_error(self) -> None:
        """Should decode the error that a path error wraps."""
        data = json.dumps(
            {
                "path": ["channel", "name"],
                "error": {"type": sy.NotFoundError.TYPE, "data": "missing"},
            }
        )
        err = create_registry().decode(
            ExceptionPayload(type=sy.PathError.TYPE, data=data)
        )
        assert isinstance(err, sy.PathError)
        assert err.path == ["channel", "name"]
        assert isinstance(err.error, sy.NotFoundError)


class TestError:
    """Tests for the base of the exception classes that Synnax defines."""

    @pytest.mark.parametrize(
        "type_",
        [
            "sy.auth",
            "sy.auth.invalid-credentials",
            "sy.auth.invalid_token",
            "sy.auth.expired_token",
            "sy.auth.session_ended",
            "sy.auth.access_denied",
            "sy.control",
            "sy.control.unauthorized",
            "sy.license",
            "sy.license.missing",
            "sy.license.expired",
            "sy.license.invalid",
            "sy.license.edition",
            "sy.license.fingerprint",
            "sy.license.too_many",
            "sy.query",
            "sy.query.not_found",
            "sy.query.multiple_results",
            "sy.route",
            "sy.unexpected",
            "sy.validation",
            "sy.unknown",
            "freighter.unreachable",
            "freighter.stream_closed",
            "freighter.eof",
            "arc",
            "unknown",
        ],
    )
    def test_decode(self, type_: str) -> None:
        """Should decode each error type that the Core sends into an Error."""
        err = create_registry().decode(ExceptionPayload(type=type_, data="refused"))
        assert isinstance(err, sy.Error)

    def test_decode_path(self) -> None:
        """Should decode a path error and its cause into Errors."""
        cause = {"type": "sy.validation", "data": "required"}
        err = create_registry().decode(
            ExceptionPayload(
                type="sy.validation.path",
                data=json.dumps({"path": ["name"], "error": cause}),
            )
        )
        assert isinstance(err, sy.PathError)
        assert isinstance(err.error, sy.Error)

    @pytest.mark.parametrize(
        "exc",
        [
            sy.ConfigurationError,
            sy.ContiguityError,
            freighter.EOF,
            freighter.StreamClosed,
            freighter.Unreachable,
        ],
    )
    def test_local(self, exc: type[Exception]) -> None:
        """Should make each Synnax exception that the client raises locally an Error."""
        assert issubclass(exc, sy.Error)
