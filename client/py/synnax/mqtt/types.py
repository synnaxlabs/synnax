#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

from pydantic import JsonValue

from synnax import device, task
from synnax.mqtt.types_gen import (
    EdgeConfig,
    EdgeTag,
    ReadConfig,
    ReadEntry,
    WriteConfig,
    WriteTarget,
)

MAKE = "mqtt"
MODEL = "MQTT broker"
INTEGRATION = "mqtt"


class ReadTask(
    task.StarterStopperMixin, task.JSONConfigMixin[ReadConfig], task.Protocol
):
    """A read task that subscribes to topics on an MQTT broker and writes the values
    in their payloads to Synnax channels.

    :param device: The key of the broker device to read from.
    :param name: A human-readable name for the task.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically.
    :param entries: The topics to read.
    """

    TYPE = "mqtt_read"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        data_saving_disabled: bool = False,
        auto_start: bool = False,
        entries: list[ReadEntry] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = ReadConfig(
            device=device,
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            entries=entries if entries is not None else [],
        )


class WriteTask(
    task.StarterStopperMixin, task.JSONConfigMixin[WriteConfig], task.Protocol
):
    """A write task that publishes a message to an MQTT broker each time a value is
    written to the command channel of a target.

    :param device: The key of the broker device to write to.
    :param name: A human-readable name for the task.
    :param auto_start: Whether to start the task automatically.
    :param targets: The topics to publish to.
    """

    TYPE = "mqtt_write"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        auto_start: bool = False,
        targets: list[WriteTarget] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            device=device,
            auto_start=auto_start,
            targets=targets if targets is not None else [],
        )


class EdgeTask(
    task.StarterStopperMixin, task.JSONConfigMixin[EdgeConfig], task.Protocol
):
    """A task that is one Sparkplug B edge node. It publishes Synnax channels as the
    tags of the edge node, and writes inbound tag commands to Synnax channels.

    :param device: The key of the broker device to publish to.
    :param name: A human-readable name for the task.
    :param group: The Sparkplug group ID of the edge node.
    :param edge_node: The Sparkplug edge node ID.
    :param authority: The control authority, from 0 to 255, of the writes that
        inbound tag commands make.
    :param auto_start: Whether to start the task automatically.
    :param tags: The tags the edge node publishes.
    """

    TYPE = "mqtt_sparkplug_edge"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        group: str = "",
        edge_node: str = "",
        authority: int = 0,
        auto_start: bool = False,
        tags: list[EdgeTag] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = EdgeConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = EdgeConfig(
            device=device,
            group=group,
            edge_node=edge_node,
            authority=authority,
            auto_start=auto_start,
            tags=tags if tags is not None else [],
        )


class Device(device.Device):
    """An MQTT broker device.

    MQTT tasks run on the driver inside the Core. Create the device on the rack of
    that driver::

        rack = client.racks.retrieve(integration=sy.mqtt.INTEGRATION)[0]
        broker = sy.mqtt.Device(host="127.0.0.1", rack=rack.key, name="Broker")

    :param host: Host name or address of the broker.
    :param port: TCP port of the broker, from 1 to 65535. None selects 1883, or 8883
        when secure.
    :param secure: Whether to connect with TLS.
    :param verification_skipped: Whether to accept any broker certificate.
    :param ca_file: Path on the Core host to the CA certificates that verify the
        broker. Empty uses the system roots.
    :param cert_file: Path on the Core host to a client certificate.
    :param key_file: Path on the Core host to the key of the client certificate.
    :param username: Optional username.
    :param password: Optional password.
    :param client_id: MQTT client ID. Empty derives one from the device key.
    :param keep_alive: MQTT keep-alive interval in seconds, from 1 to 65535. None
        selects 30.
    :param host_id: Sparkplug B host application ID. When set, the Core publishes its
        STATE message under this ID. Empty makes the Core a passive host.
    :param groups: Sparkplug B group IDs to browse. Empty browses every group.
    :param name: Human-readable name for the device.
    :param rack: Key of the rack of the driver inside the Core.
    :param key: Unique key. Auto-generated if empty.
    :param configured: Whether the device has been configured.
    :raises ValueError: If port or keep_alive is out of range.
    """

    def __init__(
        self,
        *,
        host: str,
        port: int | None = None,
        secure: bool = False,
        verification_skipped: bool = False,
        ca_file: str = "",
        cert_file: str = "",
        key_file: str = "",
        username: str = "",
        password: str = "",
        client_id: str = "",
        keep_alive: int | None = None,
        host_id: str = "",
        groups: list[str] | None = None,
        name: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = True,
    ):
        if not key:
            key = str(uuid4())
        if port is not None and not 1 <= port <= 65535:
            raise ValueError("port must be 1 to 65535")
        if keep_alive is not None and not 1 <= keep_alive <= 65535:
            raise ValueError("keep_alive must be 1 to 65535")
        sparkplug_groups: list[JsonValue] = list(groups) if groups is not None else []
        properties: dict[str, JsonValue] = {
            "secure": secure,
            "verification_skipped": verification_skipped,
            "ca_file": ca_file,
            "cert_file": cert_file,
            "key_file": key_file,
            "username": username,
            "password": password,
            "client_id": client_id,
            "sparkplug": {"host_id": host_id, "groups": sparkplug_groups},
            "version": 1,
        }
        if port is not None:
            properties["port"] = port
        if keep_alive is not None:
            properties["keep_alive"] = keep_alive
        super().__init__(
            key=key,
            location=host,
            rack=rack,
            name=name,
            make=MAKE,
            model=MODEL,
            configured=configured,
            properties=properties,
        )
