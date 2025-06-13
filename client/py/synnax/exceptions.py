#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from dataclasses import dataclass
from typing import Any, List, Union

import freighter

_FREIGHTER_EXCEPTION_PREFIX = "sy."


class ConfigurationError(Exception):
    """Raised when a configuration error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "configuration"


class ValidationError(Exception):
    """Raised when a validation error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "validation"


class ControlError(Exception):
    """Raised when a control error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "control"


class UnauthorizedError(ControlError):
    """Raised when an entity attempts to access or modify information it is not allowed."""

    TYPE = ControlError.TYPE + ".unauthorized"


class AuthError(Exception):
    """Raised when an authentication error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "auth"


class InvalidCredentials(AuthError):
    """Raised when invalid credentials are provided."""

    TYPE = AuthError.TYPE + ".invalid-credentials"


class InvalidToken(AuthError):
    """Raised when an invalid token is provided."""

    TYPE = AuthError.TYPE + ".invalid_token"


class ExpiredToken(AuthError):
    """Raised when a token has expired."""

    TYPE = AuthError.TYPE + ".expired_token"


class UnexpectedError(Exception):
    """Raised when an unexpected error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "unexpected"


class ContiguityError(Exception):
    """Raised when time-series data is not contiguous."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "contiguity"


class QueryError(Exception):
    """Raised when a query error occurs, such as an item not found."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query"


class NotFoundError(QueryError):
    """Raised when a query returns no results."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query.not_found"


class MultipleFoundError(QueryError):
    """Raised when a query that should return a single result returns multiple."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query.multiple_results"


class RouteError(Exception):
    """Raised when an API routing error occurs, such as a 404."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "route"
    path: str

    def __init__(self, path: str, *args: Any) -> None:
        super().__init__(*args)
        self.path = path


@dataclass
class PathError(ValidationError):
    """Raised when a validation error occurs on a specific path."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "validation.path"
    path: List[str]
    error: Exception

    def __init__(self, path: Union[str, List[str]], error: Exception):
        if isinstance(path, str):
            path = path.split(".")
        self.path = path
        self.error = error
        super().__init__(f"{'.'.join(path)}: {error}")


def _decode(encoded: freighter.ExceptionPayload) -> Exception | None:
    if encoded.type is None:
        return None if encoded.data is None else UnexpectedError(encoded.data)

    if not encoded.type.startswith(_FREIGHTER_EXCEPTION_PREFIX):
        return None

    if encoded.type.startswith(AuthError.TYPE):
        if encoded.type.startswith(InvalidCredentials.TYPE):
            return InvalidCredentials(encoded.data)
        return AuthError(encoded.data)

    if encoded.type.startswith(UnexpectedError.TYPE):
        return UnexpectedError(encoded.data)

    if encoded.type.startswith(ValidationError.TYPE):
        if encoded.type.startswith(PathError.TYPE):
            if encoded.data is None:
                return UnexpectedError(encoded.data)
            try:
                data = json.loads(encoded.data)
                return PathError(
                    data["path"],
                    freighter.decode_exception(
                        freighter.ExceptionPayload(**data["error"])
                    ),
                )
            except Exception as e:
                return UnexpectedError(f"Failed to decode PathError: {e}")
        return ValidationError(encoded.data)

    if encoded.type.startswith(QueryError.TYPE):
        if encoded.type.startswith(NotFoundError.TYPE):
            return NotFoundError(encoded.data)
        if encoded.type.startswith(MultipleFoundError.TYPE):
            return MultipleFoundError(encoded.data)
        return QueryError(encoded.data)

    if encoded.type.startswith(RouteError.TYPE):
        if encoded.data is None:
            return UnexpectedError(encoded.data)
        return RouteError(encoded.data)

    if encoded.type.startswith(ControlError.TYPE):
        if encoded.type.startswith(UnauthorizedError.TYPE):
            return UnauthorizedError(encoded.data)
        return ControlError(encoded.data)

    return UnexpectedError(encoded.data)


def _encode(err: Exception) -> freighter.ExceptionPayload | None:
    raise NotImplementedError


freighter.register_exception(_encode, _decode)
