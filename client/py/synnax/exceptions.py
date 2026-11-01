#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from dataclasses import dataclass
from functools import partial
from typing import Any

import freighter
from x.exceptions import Error, ExceptionPayload, Provider, Registry

_FREIGHTER_EXCEPTION_PREFIX = "sy."


class ConfigurationError(Error):
    """Raised when a configuration error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "configuration"


class ValidationError(Error):
    """Raised when a validation error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "validation"


class ControlError(Error):
    """Raised when a control error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "control"


class UnauthorizedError(ControlError):
    """Raised when an entity attempts to access or modify information it is not
    allowed.
    """

    TYPE = ControlError.TYPE + ".unauthorized"


class AuthError(Error):
    """Raised when an authentication error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "auth"


class _InvalidCredentials(AuthError):
    TYPE = AuthError.TYPE + ".invalid-credentials"


class InvalidToken(AuthError):
    """Raised when an invalid token is provided."""

    TYPE = AuthError.TYPE + ".invalid_token"


class ExpiredToken(AuthError):
    """Raised when a token has expired."""

    TYPE = AuthError.TYPE + ".expired_token"


class SessionEnded(AuthError):
    """Raised when the session a request or stream used has ended."""

    TYPE = AuthError.TYPE + ".session_ended"


class AccessDenied(AuthError):
    """Raised when a user does not have permission to perform an action."""

    TYPE = AuthError.TYPE + ".access_denied"


class LicenseError(Error):
    """Raised when the Core refuses a request over its license."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "license"


class MissingLicense(LicenseError):
    """Raised when no license is active on the Core."""

    TYPE = LicenseError.TYPE + ".missing"


class ExpiredLicense(LicenseError):
    """Raised when the license on the Core no longer applies."""

    TYPE = LicenseError.TYPE + ".expired"


class InvalidLicense(LicenseError):
    """Raised when a license key cannot be verified or is malformed."""

    TYPE = LicenseError.TYPE + ".invalid"


class LicenseFingerprintMismatch(LicenseError):
    """Raised when a license is bound to a different machine."""

    TYPE = LicenseError.TYPE + ".fingerprint"


class LicenseEditionMismatch(LicenseError):
    """Raised when a Desktop license is active on a Core that listens on an address
    other than loopback.
    """

    TYPE = LicenseError.TYPE + ".edition"


class LicenseLimitExceeded(LicenseError):
    """Raised when a channel would exceed the license's channel cap."""

    TYPE = LicenseError.TYPE + ".too_many"


class UnexpectedError(Error):
    """Raised when an unexpected error occurs."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "unexpected"


class QueryError(Error):
    """Raised when a query error occurs, such as an item not found."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query"


class NotFoundError(QueryError):
    """Raised when a query returns no results."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query.not_found"


class MultipleFoundError(QueryError):
    """Raised when a query that should return a single result returns multiple."""

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query.multiple_results"


class RouteError(Error):
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
    path: list[str]
    error: Exception

    def __init__(self, path: str | list[str], error: Exception):
        if isinstance(path, str):
            path = path.split(".")
        self.path = path
        self.error = error
        super().__init__(f"{'.'.join(path)}: {error}")


def _decode(registry: Registry, encoded: ExceptionPayload) -> Exception | None:
    if encoded.type is None:
        return None if encoded.data is None else UnexpectedError(encoded.data)

    if not encoded.type.startswith(_FREIGHTER_EXCEPTION_PREFIX):
        return None

    if encoded.type.startswith(AuthError.TYPE):
        if encoded.type.startswith(_InvalidCredentials.TYPE):
            return _InvalidCredentials(encoded.data)
        if encoded.type.startswith(InvalidToken.TYPE):
            return InvalidToken(encoded.data)
        if encoded.type.startswith(ExpiredToken.TYPE):
            return ExpiredToken(encoded.data)
        if encoded.type.startswith(SessionEnded.TYPE):
            return SessionEnded(encoded.data)
        if encoded.type.startswith(AccessDenied.TYPE):
            return AccessDenied(encoded.data)
        return AuthError(encoded.data)

    if encoded.type.startswith(UnexpectedError.TYPE):
        return UnexpectedError(encoded.data)

    if encoded.type.startswith(ValidationError.TYPE):
        if encoded.type.startswith(PathError.TYPE):
            if encoded.data is None:
                return UnexpectedError(encoded.data)
            try:
                data = json.loads(encoded.data)
                decoded_err = registry.decode(ExceptionPayload(**data["error"]))
                if decoded_err is None:
                    return UnexpectedError(encoded.data)
                return PathError(
                    data["path"],
                    decoded_err,
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

    if encoded.type.startswith(LicenseError.TYPE):
        if encoded.type.startswith(MissingLicense.TYPE):
            return MissingLicense(encoded.data)
        if encoded.type.startswith(ExpiredLicense.TYPE):
            return ExpiredLicense(encoded.data)
        if encoded.type.startswith(InvalidLicense.TYPE):
            return InvalidLicense(encoded.data)
        if encoded.type.startswith(LicenseEditionMismatch.TYPE):
            return LicenseEditionMismatch(encoded.data)
        if encoded.type.startswith(LicenseFingerprintMismatch.TYPE):
            return LicenseFingerprintMismatch(encoded.data)
        if encoded.type.startswith(LicenseLimitExceeded.TYPE):
            return LicenseLimitExceeded(encoded.data)
        return LicenseError(encoded.data)

    return UnexpectedError(encoded.data)


def _encode(_: Exception) -> ExceptionPayload | None:
    return None


def create_registry() -> Registry:
    """:returns: a registry that decodes the errors of Freighter and the Core."""
    registry = Registry()
    registry.register(freighter.EXCEPTION_PROVIDER)
    registry.register(Provider(encode=_encode, decode=partial(_decode, registry)))
    return registry
