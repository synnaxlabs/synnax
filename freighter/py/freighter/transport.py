from typing import Protocol, Any, TypeVar
from dataclasses import dataclass


class Payload(Protocol):
    """
    Payload is a piece of data that can be sent over the freighter.
    """
    __dataclass_fields__: dict


# Represents the inbound payload for a freighter.
I = TypeVar("I")
# Represents the outbound payload for a freighter.
O = TypeVar("O")


@dataclass
class Digest:
    """
    digest contains a set of attributes that briefly describe the underlying
    transport implementation.

    :param protocol: a string description of the protocol being used
    e.g. 'grpc.'
    :param encoder: a string description of the encoder being used to encode.
    transport payloads.
    """
    protocol: str
    encoder: str


class Transport(Protocol):
    """
    A protocol class representing a general network transport between two
    entities. This protocol is mainly descriptive.
    """

    def digest(self) -> Digest:
        """
        :return: the digest description of the freighter.
        """
        ...
