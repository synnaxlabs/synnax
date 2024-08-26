#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from dataclasses import dataclass
from enum import Enum

import freighter

_FREIGHTER_EXCEPTION_PREFIX = "sy."


@dataclass
class Field:
    field: str
    message: str


class ConfigurationError(Exception):
    """
    Raised when a configuration error occurs.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "configuration"

    pass


class ValidationError(Exception):
    """
    Raised when a validation error occurs.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "validation"


class FieldError(ValidationError):
    field: str

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "validation.field"

    def __init__(self, field: str, message: str):
        self.field = field
        super().__init__(f"{field}: {message}")


class ControlError(Exception):
    """
    Raised when a control error occurs.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "control"

    pass


class UnauthorizedError(ControlError):
    """
    Raised when an entity attempts to access or modify information it is not allowed.
    """

    TYPE = ControlError.TYPE + ".unauthorized"

    pass


class AuthError(Exception):
    """
    Raised when an authentication error occurs.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "auth"

    pass


class UnexpectedError(Exception):
    """
    Raised when an unexpected error occurs.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "unexpected"

    pass


class ContiguityError(Exception):
    """
    Raised when time-series data is not contiguous.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "contiguity"

    pass


class QueryError(Exception):
    """
    Raised when a query error occurs, such as an item not found.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query"

    pass


class NotFoundError(QueryError):
    """
    Raised when a query returns no results.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query.not_found"

    pass


class MultipleFoundError(QueryError):
    """
    Raised when a query that should return a single result returns multiple.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query.multiple_results"

    pass


class RouteError(Exception):
    """
    Raised when an API routing error occurs, such as a 404.
    """

    TYPE = _FREIGHTER_EXCEPTION_PREFIX + "route"

    path: str

    def __init__(self, path: str, *args):
        super().__init__(*args)
        self.path = path


def _decode(encoded: freighter.ExceptionPayload) -> Exception | None:
    if not encoded.type.startswith(_FREIGHTER_EXCEPTION_PREFIX):
        return None

    if encoded.type.startswith(AuthError.TYPE):
        return AuthError(encoded.data)

    if encoded.type.startswith(UnexpectedError.TYPE):
        return UnexpectedError(encoded.data)

    if encoded.type.startswith(ValidationError.TYPE):
        if encoded.type.startswith(FieldError.TYPE):
            values = encoded.data.split(":")
            if len(values) != 2:
                return UnexpectedError(encoded.data)
            return FieldError(values[0], values[1])
        return ValidationError(encoded.data)

    if encoded.type.startswith(QueryError.TYPE):
        if encoded.type.startswith(NotFoundError.TYPE):
            return NotFoundError(encoded.data)
        if encoded.type.startswith(MultipleFoundError.TYPE):
            return MultipleFoundError(encoded.data)
        return QueryError(encoded.data)

    if encoded.type.startswith(RouteError.TYPE):
        return RouteError(encoded.data)

    if encoded.type.startswith(ControlError.TYPE):
        if encoded.type.startswith(UnauthorizedError.TYPE):
            return UnauthorizedError(encoded.data)
        return ControlError(encoded.data)

    return UnexpectedError(encoded.data)


def _encode(err: Exception) -> freighter.ExceptionPayload | None:
    raise NotImplemented


freighter.register_exception(_encode, _decode)
