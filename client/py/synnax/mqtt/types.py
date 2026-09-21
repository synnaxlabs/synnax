#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

from synnax import device, task
from synnax.mqtt.types_gen import (
    ReadConfig,
    ReadEntry,
    WriteConfig,
    WriteTarget,
)

MAKE = "mqtt"
MODEL = "MQTT broker"
INTEGRATION = "mqtt"


def _follow_device_rack(
    internal: task.Task, device_client: device.Client, key: device.Key
) -> device.Device:
    """Puts a task that has no rack on the rack of its broker device. MQTT tasks run
    on the driver inside the Core, which only runs the tasks of its own rack."""
    dev = device_client.retrieve(key=key)
    if internal.rack == 0:
        internal.rack = dev.rack
    return dev


class ReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
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
    config: ReadConfig
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

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        return _follow_device_rack(self._internal, device_client, self.config.device)


class WriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A write task that publishes a message to an MQTT broker each time a value is
    written to the command channel of a target.

    :param device: The key of the broker device to write to.
    :param name: A human-readable name for the task.
    :param auto_start: Whether to start the task automatically.
    :param targets: The topics to publish to.
    """

    TYPE = "mqtt_write"
    config: WriteConfig
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

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        return _follow_device_rack(self._internal, device_client, self.config.device)


class Device(device.Device):
    """An MQTT broker device.

    MQTT tasks run on the driver inside the Core. Create the device on the rack of
    that driver::

        rack = client.racks.retrieve(integration=sy.mqtt.INTEGRATION)
        broker = sy.mqtt.Device(host="127.0.0.1", rack=rack.key, name="Broker")

    :param host: Host name or address of the broker.
    :param port: TCP port of the broker. Zero selects 1883, or 8883 when secure.
    :param secure: Whether to connect with TLS.
    :param verification_skipped: Whether to accept any broker certificate.
    :param ca_file: Path on the Core host to the CA certificates that verify the
        broker. Empty uses the system roots.
    :param cert_file: Path on the Core host to a client certificate.
    :param key_file: Path on the Core host to the key of the client certificate.
    :param username: Optional username.
    :param password: Optional password.
    :param client_id: MQTT client ID. Empty derives one from the device key.
    :param keep_alive: MQTT keep-alive interval in seconds. Zero selects 30.
    :param name: Human-readable name for the device.
    :param rack: Key of the rack of the driver inside the Core.
    :param key: Unique key. Auto-generated if empty.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        host: str,
        port: int = 0,
        secure: bool = False,
        verification_skipped: bool = False,
        ca_file: str = "",
        cert_file: str = "",
        key_file: str = "",
        username: str = "",
        password: str = "",
        client_id: str = "",
        keep_alive: int = 0,
        name: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = True,
    ):
        if not key:
            key = str(uuid4())
        if not 0 <= port <= 65535:
            raise ValueError("port must be 0 to 65535")
        if keep_alive < 0:
            raise ValueError("keep_alive must not be negative")
        super().__init__(
            key=key,
            location=host,
            rack=rack,
            name=name,
            make=MAKE,
            model=MODEL,
            configured=configured,
            properties={
                "port": port,
                "secure": secure,
                "verification_skipped": verification_skipped,
                "ca_file": ca_file,
                "cert_file": cert_file,
                "key_file": key_file,
                "username": username,
                "password": password,
                "client_id": client_id,
                "keep_alive": keep_alive,
                "version": 1,
            },
        )
