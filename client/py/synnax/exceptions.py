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

_FREIGHTER_EXCEPTION_PREFIX = "sy.api."


@dataclass
class Field:
    field: str
    message: str


class APIErrorType(Enum):
    GENERAL = _FREIGHTER_EXCEPTION_PREFIX + "general"
    PARSE = _FREIGHTER_EXCEPTION_PREFIX + "parse"
    AUTH = _FREIGHTER_EXCEPTION_PREFIX + "auth"
    UNEXPECTED = _FREIGHTER_EXCEPTION_PREFIX + "unexpected"
    VALIDATION = _FREIGHTER_EXCEPTION_PREFIX + "validation"
    QUERY = _FREIGHTER_EXCEPTION_PREFIX + "query"
    ROUTE = _FREIGHTER_EXCEPTION_PREFIX + "route"


class ValidationError(Exception):
    """
    Raised when a validation error occurs.
    """

    fields: list[Field]

    def __init__(self, fields_or_message: list[dict] | str | Field):
        if isinstance(fields_or_message, Field):
            self.fields = [fields_or_message]
            super(ValidationError, self).__init__(fields_or_message.message)
        elif isinstance(fields_or_message, str):
            self.fields = list()
            super(ValidationError, self).__init__(fields_or_message)
        else:
            self.fields = [Field(f["field"], f["message"]) for f in fields_or_message]
            super(ValidationError, self).__init__(self.__str__())

    def __str__(self):
        if len(self.fields) == 0:
            return super().__str__()
        return "\n".join([f"{f.field}: {f.message}" for f in self.fields])


class GeneralError(Exception):
    """
    Raised when a general error occurs.
    """

    pass


class ParseError(Exception):
    """
    Raised when a parse error occurs.
    """

    pass


class AuthError(Exception):
    """
    Raised when an authentication error occurs.
    """

    pass


class UnexpectedError(Exception):
    """
    Raised when an unexpected error occurs.
    """

    pass


class ContiguityError(Exception):
    """
    Raised when time-series data is not contiguous.
    """

    pass


class QueryError(Exception):
    """
    Raised when a query error occurs, such as an item not found.
    """

    pass


class NotFoundError(QueryError):
    """
    Raised when a query returns no results.
    """

    pass


class MultipleFoundError(QueryError):
    """
    Raised when a query that should return a single result returns multiple.
    """

    pass


class RouteError(Exception):
    """
    Raised when an API routing error occurs, such as a 404.
    """

    path: str

    def __init__(self, path: str, *args):
        super().__init__(*args)
        self.path = path


def _decode(encoded: freighter.ExceptionPayload) -> Exception | None:
    if not encoded.type.startswith(_FREIGHTER_EXCEPTION_PREFIX):
        return None

    if encoded.type == APIErrorType.GENERAL.value:
        return GeneralError(encoded.data)

    if encoded.type == APIErrorType.PARSE.value:
        return ParseError(encoded.data)

    if encoded.type == APIErrorType.AUTH.value:
        return AuthError(encoded.data)

    if encoded.type == APIErrorType.UNEXPECTED.value:
        return UnexpectedError(encoded.data)

    if encoded.type == APIErrorType.VALIDATION.value:
        return ValidationError(encoded.data)

    if encoded.type == APIErrorType.QUERY.value:
        return QueryError(encoded.data)

    if encoded.type == APIErrorType.ROUTE.value:
        return RouteError(encoded.data)

    return UnexpectedError(encoded.data)


def _encode(err: Exception) -> freighter.ExceptionPayload | None:
    raise NotImplemented


freighter.register_exception(_encode, _decode)
