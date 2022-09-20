from __future__ import annotations

from dataclasses import dataclass

from freighter import UnaryClient
from synnax.transport import Transport

from .record import ChannelRecord


@dataclass
class _Request:
    keys: list[str] | None = None
    node_id: int = None
    names: list[str] | None = None


@dataclass
class _Response:
    channels: list[ChannelRecord]

    def load(self, data: dict):
        self.channels = [ChannelRecord(**c) for c in data["channels"]]

    @classmethod
    def new(cls):
        return _Response(channels=[])


class Retrieve:
    transport: UnaryClient[_Request, _Response]

    def __init__(self, transport: Transport):
        self.transport = transport.http.get(_Request, _Response)
