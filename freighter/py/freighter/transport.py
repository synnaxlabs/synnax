from typing import Protocol, Any, TypeVar
from dataclasses import dataclass


class Payload(Protocol):
    """
    Payload is a piece of data that can be sent over the freighter.
    """

    __dataclass_fields__: dict


# Represents the inbound payload for a freighter.
RS = TypeVar("RS", bound=Payload)
# Represents the outbound payload for a freighter.
RQ = TypeVar("RQ", bound=Payload)


class Transport(Protocol):
    """
    A protocol class representing a general network transport between two
    entities. This protocol is mainly descriptive.
    """
