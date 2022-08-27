from typing import Protocol, Any, TypeVar, Callable, Generic
from dataclasses import dataclass


class Payload(Protocol):
    """
    Payload is a piece of data that can be sent over the freighter.
    """

    __dataclass_fields__: dict


# Represents the inbound payload for a freighter.
RS = TypeVar("RS", bound=Payload, covariant=True)
# Represents the outbound payload for a freighter.
RQ = TypeVar("RQ", bound=Payload, contravariant=True)
# Represents any payload.
P = TypeVar("P", bound=Payload)


class Transport(Protocol):
    """
    A protocol class representing a general network transport between two
    entities. This protocol is mainly descriptive.
    """


PayloadFactoryFunc = Callable[[], P]


class PayloadFactory(Generic[P]):
    _factory: PayloadFactoryFunc[P] | None

    def __init__(self, factory: PayloadFactoryFunc[P]):
        self._factory = factory

    def __call__(self) -> P:
        assert self._factory is not None
        return self._factory()
