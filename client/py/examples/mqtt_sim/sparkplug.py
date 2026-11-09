#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A mock Sparkplug B edge node and host for the mock MQTT plant.

The payloads use ``sparkplug_b_pb2``, which is generated from the schema that the Core
uses. See the README to generate it again.
"""

import json
import time
from dataclasses import dataclass

from google.protobuf.message import DecodeError

from examples.mqtt_sim import sparkplug_b_pb2 as pb
from examples.mqtt_sim.broker import Broker

NAMESPACE = "spBv1.0"
REBIRTH = "Node Control/Rebirth"
BD_SEQ = "bdSeq"

Value = float | int | bool | str

# Sparkplug B puts integers of up to 32 bits in int_value and wider ones in long_value,
# with signed integers in two's complement.
_INT_TYPES: set[int] = {pb.Int8, pb.Int16, pb.Int32, pb.UInt8, pb.UInt16, pb.UInt32}
_LONG_TYPES: set[int] = {pb.Int64, pb.UInt64}
_SIGNED_BITS: dict[int, int] = {pb.Int8: 8, pb.Int16: 16, pb.Int32: 32, pb.Int64: 64}


def new_metric(
    data_type: int,
    value: Value,
    *,
    name: str | None = None,
    alias: int | None = None,
    timestamp: int | None = None,
) -> pb.Payload.Metric:
    """Returns a metric that holds value as data_type.

    :param timestamp: Milliseconds since the epoch.
    :raises ValueError: If data_type is not a supported scalar type, or value does not
        convert to it.
    :raises TypeError: If value is of a type that has no conversion to data_type.
    """
    m = pb.Payload.Metric(
        name=name, alias=alias, timestamp=timestamp, datatype=data_type
    )
    if data_type in _INT_TYPES:
        m.int_value = int(value) & 0xFFFF_FFFF
    elif data_type in _LONG_TYPES:
        m.long_value = int(value) & 0xFFFF_FFFF_FFFF_FFFF
    elif data_type == pb.Float:
        m.float_value = float(value)
    elif data_type == pb.Double:
        m.double_value = float(value)
    elif data_type == pb.Boolean:
        m.boolean_value = bool(value)
    elif data_type == pb.String:
        m.string_value = str(value)
    else:
        raise ValueError(f"unsupported data type {data_type}")
    return m


def metric_value(m: pb.Payload.Metric) -> Value | None:
    """Returns the scalar value of m, or None when it holds no scalar value."""
    which = m.WhichOneof("value")
    if which == "int_value" or which == "long_value":
        raw = m.int_value if which == "int_value" else m.long_value
        bits = _SIGNED_BITS.get(m.datatype)
        if bits is None:
            return raw
        raw &= (1 << bits) - 1
        return raw - (1 << bits) if raw >> (bits - 1) else raw
    if which == "float_value":
        return m.float_value
    if which == "double_value":
        return m.double_value
    if which == "boolean_value":
        return m.boolean_value
    if which == "string_value":
        return m.string_value
    return None


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
        *,
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
        self, device: str, *, birth: bool, names: set[str] | None
    ) -> list[pb.Payload.Metric]:
        now = _now_ms()
        return [
            new_metric(
                tag.data_type,
                tag.value,
                # A data message names its tag by alias only.
                name=tag.name if birth else None,
                alias=self._aliases[device, tag.name],
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
            metrics = self._metrics(device, birth=True, names=None)
            if device == "":
                metrics = [
                    new_metric(pb.Int64, self._bd_seq, name=BD_SEQ),
                    new_metric(pb.Boolean, False, name=REBIRTH),
                    *metrics,
                ]
            payload = pb.Payload(timestamp=now, metrics=metrics, seq=self._next_seq())
            self.broker.publish(
                self.topic("DBIRTH" if device else "NBIRTH", device),
                payload.SerializeToString(),
            )

    def death(self) -> None:
        """Publishes the death of the edge node."""
        bd_seq = new_metric(pb.Int64, max(self._bd_seq, 0), name=BD_SEQ)
        payload = pb.Payload(timestamp=_now_ms(), metrics=[bd_seq])
        self.broker.publish(self.topic("NDEATH"), payload.SerializeToString())

    def data(self, device: str = "", *, names: set[str] | None = None) -> None:
        """Publishes the values of tags of device, or of all of them with no names."""
        payload = pb.Payload(
            timestamp=_now_ms(),
            metrics=self._metrics(device, birth=False, names=names),
            seq=self._next_seq(),
        )
        self.broker.publish(
            self.topic("DDATA" if device else "NDATA", device),
            payload.SerializeToString(),
        )

    def _on_message(self, topic: str, payload: bytes) -> None:
        for device in self.tags:
            if topic == self.topic("DCMD" if device else "NCMD", device):
                break
        else:
            return
        try:
            command = pb.Payload.FromString(payload)
        except DecodeError:
            return
        changed: set[str] = set()
        for metric in command.metrics:
            value = metric_value(metric)
            if device == "" and metric.name == REBIRTH:
                if value is True:
                    self.birth()
                continue
            for tag in self.tags[device]:
                if tag.name == metric.name and value is not None:
                    tag.value = value
                    changed.add(tag.name)
        if changed:
            self.data(device, names=changed)


MIRROR_PREFIX = "plant/sparkplug/"
MIRROR_SUFFIX = "/set"


class HostMirror:
    """A Sparkplug B host that mirrors the tags of every edge node on plain topics.

    Each tag value is published, retained, as JSON ``{"value": ...}`` on
    ``plant/sparkplug/<group>/<edge node>/<tag>``. A JSON number or string published on
    that topic plus ``/set`` becomes an NCMD for the tag. The mirror lets a plain MQTT
    client see and command the edge nodes of a broker.

    :param broker: The broker to listen on and publish through.
    """

    def __init__(self, broker: Broker) -> None:
        self.broker = broker
        # The tags of each edge node by group and edge node ID, from its last birth.
        self._tags: dict[tuple[str, str], dict[str, pb.Payload.Metric]] = {}
        self._by_alias: dict[tuple[str, str], dict[int, str]] = {}
        broker.on_message(self._on_message)

    def _on_message(self, topic: str, payload: bytes) -> None:
        if topic.startswith(MIRROR_PREFIX) and topic.endswith(MIRROR_SUFFIX):
            self._command(topic[len(MIRROR_PREFIX) : -len(MIRROR_SUFFIX)], payload)
            return
        levels = topic.split("/")
        if len(levels) != 4 or levels[0] != NAMESPACE:
            return
        group, message_type, edge_node = levels[1], levels[2], levels[3]
        if message_type not in ("NBIRTH", "NDATA"):
            return
        try:
            decoded = pb.Payload.FromString(payload)
        except DecodeError:
            return
        node = (group, edge_node)
        if message_type == "NBIRTH":
            self._tags[node] = {}
            self._by_alias[node] = {}
        tags = self._tags.setdefault(node, {})
        by_alias = self._by_alias.setdefault(node, {})
        for m in decoded.metrics:
            if message_type == "NBIRTH":
                if m.name in (BD_SEQ, REBIRTH):
                    continue
                if m.HasField("alias"):
                    by_alias[m.alias] = m.name
                tags[m.name] = m
            else:
                name = m.name or by_alias.get(m.alias, "")
                if name not in tags:
                    continue
                m.name = name
                tags[name] = m
            self.broker.publish(
                f"{MIRROR_PREFIX}{group}/{edge_node}/{m.name}",
                json.dumps({"value": metric_value(m)}),
                retain=True,
            )

    def _command(self, path: str, payload: bytes) -> None:
        levels = path.split("/", 2)
        if len(levels) != 3:
            return
        group, edge_node, name = levels
        tag = self._tags.get((group, edge_node), {}).get(name)
        if tag is None:
            return
        try:
            metric = new_metric(tag.datatype, json.loads(payload), name=name)
        except (TypeError, ValueError):
            return
        command = pb.Payload(timestamp=_now_ms(), metrics=[metric])
        self.broker.publish(
            f"{NAMESPACE}/{group}/NCMD/{edge_node}", command.SerializeToString()
        )
