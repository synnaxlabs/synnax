#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A mock Sparkplug B edge node for the mock MQTT plant.

The payload codec covers only the fields that the edge node uses: the sequence number
and timestamp of a payload, and the name, alias, timestamp, data type, and scalar value
of a metric.
"""

import struct
import time
from dataclasses import dataclass, field

from examples.mqtt_sim.broker import Broker

NAMESPACE = "spBv1.0"
REBIRTH = "Node Control/Rebirth"
BD_SEQ = "bdSeq"

INT32 = 3
INT64 = 4
UINT64 = 8
FLOAT = 9
DOUBLE = 10
BOOLEAN = 11
STRING = 12

Value = float | int | bool | str

_VARINT, _FIXED64, _BYTES, _FIXED32 = 0, 1, 2, 5


def _varint(value: int) -> bytes:
    out = bytearray()
    while True:
        digit = value & 0x7F
        value >>= 7
        if value:
            out.append(digit | 0x80)
        else:
            out.append(digit)
            return bytes(out)


def _field(number: int, wire_type: int, body: bytes) -> bytes:
    return _varint(number << 3 | wire_type) + body


def _bytes_field(number: int, body: bytes) -> bytes:
    return _field(number, _BYTES, _varint(len(body)) + body)


def _read_varint(data: bytes, pos: int) -> tuple[int, int]:
    value = shift = 0
    while True:
        digit = data[pos]
        pos += 1
        value |= (digit & 0x7F) << shift
        if not digit & 0x80:
            return value, pos
        shift += 7


def _read_fields(data: bytes) -> list[tuple[int, int | bytes]]:
    """Returns the number and the raw value of each field of a message."""
    fields: list[tuple[int, int | bytes]] = []
    pos = 0
    while pos < len(data):
        tag, pos = _read_varint(data, pos)
        number, wire_type = tag >> 3, tag & 7
        if wire_type == _VARINT:
            value, pos = _read_varint(data, pos)
            fields.append((number, value))
            continue
        if wire_type == _BYTES:
            size, pos = _read_varint(data, pos)
        elif wire_type == _FIXED64:
            size = 8
        elif wire_type == _FIXED32:
            size = 4
        else:
            raise ValueError(f"unsupported wire type {wire_type}")
        fields.append((number, data[pos : pos + size]))
        pos += size
    return fields


@dataclass
class Metric:
    """One metric of a Sparkplug B payload."""

    name: str = ""
    alias: int | None = None
    data_type: int = 0
    value: Value | None = None
    timestamp: int | None = None
    """Milliseconds since the epoch."""

    def encode(self) -> bytes:
        out = bytearray()
        if self.name:
            out += _bytes_field(1, self.name.encode())
        if self.alias is not None:
            out += _field(2, _VARINT, _varint(self.alias))
        if self.timestamp is not None:
            out += _field(3, _VARINT, _varint(self.timestamp))
        out += _field(4, _VARINT, _varint(self.data_type))
        if self.data_type == INT32:
            out += _field(10, _VARINT, _varint(int(self.value or 0) & 0xFFFFFFFF))
        elif self.data_type in (INT64, UINT64):
            out += _field(11, _VARINT, _varint(int(self.value or 0) & (2**64 - 1)))
        elif self.data_type == FLOAT:
            out += _field(12, _FIXED32, struct.pack("<f", self.value))
        elif self.data_type == DOUBLE:
            out += _field(13, _FIXED64, struct.pack("<d", self.value))
        elif self.data_type == BOOLEAN:
            out += _field(14, _VARINT, _varint(int(bool(self.value))))
        elif self.data_type == STRING:
            out += _bytes_field(15, str(self.value).encode())
        else:
            raise ValueError(f"unsupported data type {self.data_type}")
        return bytes(out)

    @staticmethod
    def decode(data: bytes) -> "Metric":
        m = Metric()
        for number, raw in _read_fields(data):
            if number == 1 and isinstance(raw, bytes):
                m.name = raw.decode()
            elif number == 2 and isinstance(raw, int):
                m.alias = raw
            elif number == 3 and isinstance(raw, int):
                m.timestamp = raw
            elif number == 4 and isinstance(raw, int):
                m.data_type = raw
            elif number == 10 and isinstance(raw, int):
                m.value = raw - 2**32 if raw >= 2**31 else raw
            elif number == 11 and isinstance(raw, int):
                m.value = raw
            elif number == 12 and isinstance(raw, bytes):
                m.value = struct.unpack("<f", raw)[0]
            elif number == 13 and isinstance(raw, bytes):
                m.value = struct.unpack("<d", raw)[0]
            elif number == 14 and isinstance(raw, int):
                m.value = bool(raw)
            elif number == 15 and isinstance(raw, bytes):
                m.value = raw.decode()
        if m.data_type == INT64 and isinstance(m.value, int) and m.value >= 2**63:
            m.value -= 2**64
        return m


@dataclass
class Payload:
    """A Sparkplug B payload."""

    metrics: list[Metric] = field(default_factory=list)
    seq: int | None = None
    timestamp: int | None = None
    """Milliseconds since the epoch."""

    def encode(self) -> bytes:
        out = bytearray()
        if self.timestamp is not None:
            out += _field(1, _VARINT, _varint(self.timestamp))
        for m in self.metrics:
            out += _bytes_field(2, m.encode())
        if self.seq is not None:
            out += _field(3, _VARINT, _varint(self.seq))
        return bytes(out)

    @staticmethod
    def decode(data: bytes) -> "Payload":
        p = Payload()
        for number, raw in _read_fields(data):
            if number == 1 and isinstance(raw, int):
                p.timestamp = raw
            elif number == 2 and isinstance(raw, bytes):
                p.metrics.append(Metric.decode(raw))
            elif number == 3 and isinstance(raw, int):
                p.seq = raw
        return p


@dataclass
class Tag:
    """One tag of an edge node or a device."""

    name: str
    data_type: int
    value: Value


def _now_ms() -> int:
    return time.time_ns() // 1_000_000


class EdgeNode:
    """A Sparkplug B edge node that publishes through a broker in the same process.

    It answers a rebirth request with its birth messages, and it takes each command for
    one of its tags as the new value of that tag.

    :param broker: The broker to publish through.
    :param group: The group ID.
    :param edge_node: The edge node ID.
    :param tags: The tags of the edge node.
    :param devices: The tags of each device, by device ID.
    """

    def __init__(
        self,
        broker: Broker,
        group: str,
        edge_node: str,
        tags: list[Tag],
        devices: dict[str, list[Tag]] | None = None,
    ) -> None:
        self.broker = broker
        self.group = group
        self.edge_node = edge_node
        # The tags of the edge node sit under the empty device ID.
        self.tags: dict[str, list[Tag]] = {"": tags, **(devices or {})}
        self._aliases: dict[tuple[str, str], int] = {}
        for device, device_tags in self.tags.items():
            for tag in device_tags:
                self._aliases[device, tag.name] = len(self._aliases) + 1
        self._seq = 0
        self._bd_seq = -1
        broker.on_message(self._on_message)

    def topic(self, message_type: str, device: str = "") -> str:
        topic = f"{NAMESPACE}/{self.group}/{message_type}/{self.edge_node}"
        return f"{topic}/{device}" if device else topic

    def _next_seq(self) -> int:
        seq = self._seq
        self._seq = (self._seq + 1) % 256
        return seq

    def _metrics(
        self, device: str, birth: bool, names: set[str] | None
    ) -> list[Metric]:
        now = _now_ms()
        return [
            Metric(
                # A data message names its tag by alias only.
                name=tag.name if birth else "",
                alias=self._aliases[device, tag.name],
                data_type=tag.data_type,
                value=tag.value,
                timestamp=now,
            )
            for tag in self.tags[device]
            if names is None or tag.name in names
        ]

    def birth(self) -> None:
        """Publishes the birth of the edge node and of each of its devices."""
        self._seq = 0
        self._bd_seq = (self._bd_seq + 1) % 256
        now = _now_ms()
        for device in self.tags:
            metrics = self._metrics(device, True, None)
            if device == "":
                metrics = [
                    Metric(name=BD_SEQ, data_type=INT64, value=self._bd_seq),
                    Metric(name=REBIRTH, data_type=BOOLEAN, value=False),
                    *metrics,
                ]
            self.broker.publish(
                self.topic("DBIRTH" if device else "NBIRTH", device),
                Payload(metrics, self._next_seq(), now).encode(),
            )

    def death(self) -> None:
        """Publishes the death of the edge node."""
        bd_seq = Metric(name=BD_SEQ, data_type=INT64, value=max(self._bd_seq, 0))
        self.broker.publish(
            self.topic("NDEATH"), Payload([bd_seq], timestamp=_now_ms()).encode()
        )

    def data(self, device: str = "", names: set[str] | None = None) -> None:
        """Publishes the values of tags of device, or of all of them with no names."""
        self.broker.publish(
            self.topic("DDATA" if device else "NDATA", device),
            Payload(
                self._metrics(device, False, names), self._next_seq(), _now_ms()
            ).encode(),
        )

    def _on_message(self, topic: str, payload: bytes) -> None:
        for device in self.tags:
            if topic == self.topic("DCMD" if device else "NCMD", device):
                break
        else:
            return
        try:
            command = Payload.decode(payload)
        except (ValueError, IndexError, struct.error, UnicodeDecodeError):
            return
        changed: set[str] = set()
        for metric in command.metrics:
            if device == "" and metric.name == REBIRTH:
                if metric.value is True:
                    self.birth()
                continue
            for tag in self.tags[device]:
                if tag.name == metric.name and metric.value is not None:
                    tag.value = metric.value
                    changed.add(tag.name)
        if changed:
            self.data(device, changed)
