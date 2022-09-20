from __future__ import annotations

from typing import Protocol, Type, TypeVar


class Payload(Protocol):
    """
    Payload is a piece of data that can be sent over the freighter.
    """

    __dataclass_fields__: dict


# Represents the inbound payload for a freighter.
RS = TypeVar("RS", bound="Response", covariant=True)


class Response(Payload):
    """
    Response is a piece of data that can be received over the freighter.
    """

    @classmethod
    def new(cls: Type[RS]) -> RS:
        """Creates a new instance of the Response with any blank fields correctly
        initialized.
        :returns: A new instance of the Response.
        """
        ...


# Represents the outbound payload for a freighter.
RQ = TypeVar("RQ", bound=Payload, contravariant=True)
# Represents any payload.
P = TypeVar("P", bound=Payload)
