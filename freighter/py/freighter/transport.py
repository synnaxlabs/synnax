from __future__ import annotations

from typing import TypeAlias, TypeVar

from pydantic import BaseModel

Payload: TypeAlias = BaseModel

# Represents the inbound payload for a freighter.
RS = TypeVar("RS", bound=Payload, covariant=True)
# Represents the outbound payload for a freighter.
RQ = TypeVar("RQ", bound=Payload, contravariant=True)
# Represents any payload.
P = TypeVar("P", bound=Payload)
