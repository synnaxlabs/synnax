import json
from dataclasses import dataclass
from enum import Enum

import freighter

_FREIGHTER_EXCEPTION_TYPE = "delta.api.errors"


@dataclass
class APIExceptionPayload:
    type: str | None
    error: dict


class APIErrorType(Enum):
    GENERAL = "general"
    NIL = "nil"
    PARSE = "parse"
    AUTH = "auth"
    UNEXPECTED = "unexpected"
    VALIDATION = "validation"
    QUERY = "query"
    ROUTE = "route"


class ValidationField:
    def __init__(self, field: str, message: str):
        self.field = field
        self.message = message


class ValidationError(Exception):
    """
    Raised when a validation error occurs.
    """

    fields: list[ValidationField]

    def __init__(self, fieldsOrMessage: list[dict] | str | ValidationField):
        if isinstance(fieldsOrMessage, ValidationField):
            self.fields = [fieldsOrMessage]
            super(ValidationError, self).__init__(fieldsOrMessage.message)
        elif isinstance(fieldsOrMessage, str):
            super(ValidationError, self).__init__(fieldsOrMessage)
        else:
            self.fields = [
                ValidationField(f["field"], f["message"]) for f in fieldsOrMessage
            ]
            super(ValidationError, self).__init__(self.__str__())

    def __str__(self):
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


class RouteError(Exception):
    """
    Raised when an API routing error occurs, such as a 404.
    """

    path: str

    def __init__(self, path: str, *args):
        super().__init__(*args)
        self.path = path


@dataclass
class Field:
    field: str
    message: str


def maybe_raise_from_res(res: dict):
    """
    Raise an error from a dictionary response.
    """
    exc = parse_from_payload(res)
    if exc is not None:
        raise exc


def parse_from_payload(pld: APIExceptionPayload) -> Exception | None:
    """
    Parse an error from a dictionary response.
    """

    if type(pld) == dict:
        raise UnexpectedError(f"Unknown error type {pld}")

    if pld.type is None:
        raise ValueError(f"{pld} is not a valid error payload")

    if pld.type == APIErrorType.NIL.value:
        return None

    if pld.error is None:
        raise ValueError(f"{pld} is not a valid error payload")

    if pld.type == APIErrorType.GENERAL.value:
        return GeneralError(pld.error["message"])

    if pld.type == APIErrorType.PARSE.value:
        return ParseError(pld.error["message"])

    if pld.type == APIErrorType.AUTH.value:
        return AuthError(pld.error["message"])

    if pld.type == APIErrorType.UNEXPECTED.value:
        return UnexpectedError(pld.error)

    if pld.type == APIErrorType.VALIDATION.value:
        return ValidationError(pld.error)

    if pld.type == APIErrorType.QUERY.value:
        return QueryError(pld.error["message"])

    if pld.type == APIErrorType.ROUTE.value:
        return RouteError(pld.error["path"], pld.error["message"])

    return Exception("unable to parse error")


def _decode(encoded: str) -> Exception:
    dct = json.loads(encoded)
    pld = APIExceptionPayload(**dct)
    return parse_from_payload(pld)


def _encode(err: Exception) -> str:
    raise NotImplemented


freighter.register_exception(_FREIGHTER_EXCEPTION_TYPE, _encode, _decode)
