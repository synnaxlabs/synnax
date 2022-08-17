import json
from dataclasses import dataclass
from enum import Enum

import freighter.errors

_FREIGHTER_ERROR_TYPE = "delta.api.errors"


@dataclass
class APIErrorPayload:
    type: str | None
    error: dict


class APIErrorType(Enum):
    GENERAL = "general"
    NIL = "nil"
    PARSE = "parse"
    AUTH = "auth"
    UNEXPECTED = "unexpected"
    VALIDATION = "validation"


class ValidationError(Exception):
    """
    Raised when a validation error occurs.
    """
    pass


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


@dataclass
class Field:
    field: str
    message: str


class TransportClosed(Exception):
    """
    Raised when the transport is closed.
    """
    pass


def maybe_raise_from_res(res: dict):
    """
    Raise an error from a dictionary response.
    """
    exc = parse_from_payload(res)
    if exc is not None:
        raise exc


def parse_from_payload(pld: APIErrorPayload) -> Exception | None:
    """
    Parse an error from a dictionary response.
    """

    print(pld)

    if type(pld) == dict:
        raise UnexpectedError(f"Unknown error type {pld}")

    if pld.type is None:
        raise ValueError(f"{pld} is not a valid error payload")

    if pld.type == APIErrorType.NIL.value:
        return None

    if pld.error is None:
        raise ValueError(f"{pld} is not a valid error payload")

    if pld.type == APIErrorType.GENERAL.value:
        return GeneralError(pld.error['message'])

    if pld.type == APIErrorType.PARSE.value:
        return ParseError(pld.error['message'])

    if pld.type == APIErrorType.AUTH.value:
        return AuthError(pld.error['message'])

    if pld.type == APIErrorType.UNEXPECTED.value:
        return UnexpectedError(pld.error)

    if pld.type == APIErrorType.VALIDATION.value:
        return ValidationError(pld.error)

    return Exception("unable to parse error")


def _decode(encoded: str) -> Exception:
    dct = json.loads(encoded)
    pld = APIErrorPayload(**dct)
    return parse_from_payload(pld)


def _encode(err: Exception) -> str:
    raise NotImplemented


freighter.errors.register(_FREIGHTER_ERROR_TYPE, _encode, _decode)
