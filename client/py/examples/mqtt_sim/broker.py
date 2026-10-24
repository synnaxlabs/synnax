#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A small MQTT 3.1.1 broker for examples and tests.

It supports what the Synnax MQTT integration uses: CONNECT, SUBSCRIBE with wildcards,
UNSUBSCRIBE, PUBLISH at every quality of service, retained messages, and keep-alive
pings. It accepts any credentials, delivers every message at quality of service 0, and
keeps no session state. Use a real broker such as Mosquitto in production.
"""

import asyncio
from collections.abc import Callable

CONNECT = 1
CONNACK = 2
PUBLISH = 3
PUBACK = 4
PUBREC = 5
PUBREL = 6
PUBCOMP = 7
SUBSCRIBE = 8
SUBACK = 9
UNSUBSCRIBE = 10
UNSUBACK = 11
PINGREQ = 12
PINGRESP = 13
DISCONNECT = 14

MessageHandler = Callable[[str, bytes], None]


def topic_matches(topic_filter: str, topic: str) -> bool:
    """Reports whether topic matches an MQTT topic filter."""
    if topic.startswith("$") and topic_filter[0] in "+#":
        return False
    filter_levels = topic_filter.split("/")
    topic_levels = topic.split("/")
    for i, level in enumerate(filter_levels):
        if level == "#":
            return True
        if i >= len(topic_levels):
            return False
        if level != "+" and level != topic_levels[i]:
            return False
    return len(filter_levels) == len(topic_levels)


def encode_length(length: int) -> bytes:
    out = bytearray()
    while True:
        digit = length % 128
        length //= 128
        if length > 0:
            digit |= 0x80
        out.append(digit)
        if length == 0:
            return bytes(out)


def encode_string(value: str) -> bytes:
    raw = value.encode()
    return len(raw).to_bytes(2, "big") + raw


def packet(kind: int, flags: int, body: bytes) -> bytes:
    return bytes([kind << 4 | flags]) + encode_length(len(body)) + body


def publish_packet(topic: str, payload: bytes, retain: bool) -> bytes:
    return packet(PUBLISH, int(retain), encode_string(topic) + payload)


class _Reader:
    """Reads the fields of one packet body."""

    def __init__(self, body: bytes) -> None:
        self.body = body
        self.pos = 0

    def done(self) -> bool:
        return self.pos >= len(self.body)

    def byte(self) -> int:
        self.pos += 1
        return self.body[self.pos - 1]

    def uint16(self) -> int:
        self.pos += 2
        return int.from_bytes(self.body[self.pos - 2 : self.pos], "big")

    def binary(self) -> bytes:
        length = self.uint16()
        self.pos += length
        return self.body[self.pos - length : self.pos]

    def string(self) -> str:
        return self.binary().decode()

    def rest(self) -> bytes:
        out = self.body[self.pos :]
        self.pos = len(self.body)
        return out


class _Session:
    """One connected client."""

    def __init__(self, writer: asyncio.StreamWriter) -> None:
        self.writer = writer
        self.client_id = ""
        self.filters: set[str] = set()
        # The IDs of the exactly-once messages that wait for their release.
        self.unreleased: set[int] = set()

    def send(self, data: bytes) -> None:
        if not self.writer.is_closing():
            self.writer.write(data)


class Broker:
    """An MQTT 3.1.1 broker that runs on the asyncio event loop of its caller.

    :param host: The address to listen on.
    :param port: The TCP port to listen on.
    """

    def __init__(self, host: str = "127.0.0.1", port: int = 1883) -> None:
        self.host = host
        self.port = port
        self._sessions: dict[str, _Session] = {}
        self._retained: dict[str, bytes] = {}
        self._handlers: list[MessageHandler] = []
        self._server: asyncio.Server | None = None

    def on_message(self, handler: MessageHandler) -> None:
        """Calls handler with the topic and payload of each message a client sends."""
        self._handlers.append(handler)

    def publish(self, topic: str, payload: bytes | str, retain: bool = False) -> None:
        """Delivers a message to every subscriber, as if a client had sent it."""
        if isinstance(payload, str):
            payload = payload.encode()
        if retain:
            if payload:
                self._retained[topic] = payload
            else:
                self._retained.pop(topic, None)
        data = publish_packet(topic, payload, False)
        for session in list(self._sessions.values()):
            if any(topic_matches(f, topic) for f in session.filters):
                session.send(data)

    async def start(self) -> None:
        self._server = await asyncio.start_server(self._serve, self.host, self.port)

    async def serve_forever(self) -> None:
        await self.start()
        assert self._server is not None
        async with self._server:
            await self._server.serve_forever()

    async def stop(self) -> None:
        if self._server is None:
            return
        self._server.close()
        for session in list(self._sessions.values()):
            session.writer.close()
        await self._server.wait_closed()
        self._server = None

    async def _serve(
        self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter
    ) -> None:
        session = _Session(writer)
        try:
            while True:
                kind, flags, body = await self._read_packet(reader)
                if kind == DISCONNECT:
                    return
                self._handle(session, kind, flags, _Reader(body))
                await writer.drain()
        except (asyncio.IncompleteReadError, ConnectionError, ValueError, IndexError):
            return
        finally:
            if self._sessions.get(session.client_id) is session:
                del self._sessions[session.client_id]
            writer.close()

    @staticmethod
    async def _read_packet(reader: asyncio.StreamReader) -> tuple[int, int, bytes]:
        first = (await reader.readexactly(1))[0]
        length, shift = 0, 0
        while True:
            digit = (await reader.readexactly(1))[0]
            length |= (digit & 0x7F) << shift
            if not digit & 0x80:
                break
            shift += 7
            if shift > 21:
                raise ValueError("malformed remaining length")
        return first >> 4, first & 0x0F, await reader.readexactly(length)

    def _handle(self, session: _Session, kind: int, flags: int, r: _Reader) -> None:
        if kind == CONNECT:
            self._connect(session, r)
        elif session.client_id == "":
            raise ValueError("the first packet must be CONNECT")
        elif kind == PUBLISH:
            self._publish(session, flags, r)
        elif kind == PUBREL:
            packet_id = r.uint16()
            session.unreleased.discard(packet_id)
            session.send(packet(PUBCOMP, 0, packet_id.to_bytes(2, "big")))
        elif kind == SUBSCRIBE:
            self._subscribe(session, r)
        elif kind == UNSUBSCRIBE:
            packet_id = r.uint16()
            while not r.done():
                session.filters.discard(r.string())
            session.send(packet(UNSUBACK, 0, packet_id.to_bytes(2, "big")))
        elif kind == PINGREQ:
            session.send(packet(PINGRESP, 0, b""))

    def _connect(self, session: _Session, r: _Reader) -> None:
        r.string()  # protocol name
        r.byte()  # protocol level
        r.byte()  # connect flags
        r.uint16()  # keep-alive
        session.client_id = r.string() or f"anonymous-{id(session)}"
        previous = self._sessions.get(session.client_id)
        if previous is not None:
            # A broker allows one session for each client ID.
            previous.writer.close()
        self._sessions[session.client_id] = session
        session.send(packet(CONNACK, 0, bytes([0, 0])))

    def _publish(self, session: _Session, flags: int, r: _Reader) -> None:
        qos = (flags >> 1) & 0x03
        retain = bool(flags & 0x01)
        topic = r.string()
        packet_id = r.uint16() if qos > 0 else 0
        payload = r.rest()
        if qos == 1:
            session.send(packet(PUBACK, 0, packet_id.to_bytes(2, "big")))
        elif qos == 2:
            session.send(packet(PUBREC, 0, packet_id.to_bytes(2, "big")))
            if packet_id in session.unreleased:
                return
            session.unreleased.add(packet_id)
        self.publish(topic, payload, retain)
        for handler in self._handlers:
            handler(topic, payload)

    def _subscribe(self, session: _Session, r: _Reader) -> None:
        packet_id = r.uint16()
        granted = bytearray()
        added: list[str] = []
        while not r.done():
            added.append(r.string())
            r.byte()  # requested quality of service
            granted.append(0)
        session.filters.update(added)
        session.send(packet(SUBACK, 0, packet_id.to_bytes(2, "big") + bytes(granted)))
        for topic, payload in self._retained.items():
            if any(topic_matches(f, topic) for f in added):
                session.send(publish_packet(topic, payload, True))
