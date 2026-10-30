#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal
from uuid import uuid4

from pydantic import JsonValue

from synnax import channel, device, task
from synnax.opcua.types_gen import (
    EventConfig,
    EventField,
    NodeWriteChannel,
    ReadChannel,
    ReadConfig,
    WriteChannel,
    WriteConfig,
)
from synnax.telem import CrudeRate, Rate

SecurityMode = Literal["None", "Sign", "SignAndEncrypt"]

SecurityPolicy = Literal[
    "None",
    "Basic128Rsa15",
    "Basic256",
    "Basic256Sha256",
    "Aes128_Sha256_RsaOaep",
    "Aes256_Sha256_RsaPss",
]


class ReadTask(
    task.StarterStopperMixin, task.JSONConfigMixin[ReadConfig], task.Protocol
):
    """A read task for sampling data from OPC UA devices and writing the data to a
    Synnax cluster. For detailed information on configuring and operating an OPC UA
    read task, see https://docs.synnaxlabs.com/reference/hardware/opc-ua/read-task

    :param device: The key of the Synnax OPC UA device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the OPC UA device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster. Only relevant when array_mode is False.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param array_mode: Whether to sample data in array mode, where each read returns an
        array of array_size samples per node.
    :param array_size: The number of samples in each array when array_mode is True.
    :param channels: The OPC UA nodes to read from and the Synnax channels to write
        their data to.
    :param nan_fill_enabled: Whether to write NaN for a bad value of a float channel.
        When False, a bad value skips the sample for every channel on its index.
    """

    TYPE = "opc_read"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        sample_rate: CrudeRate = 10,
        stream_rate: CrudeRate = 5,
        data_saving_disabled: bool = False,
        auto_start: bool = False,
        array_mode: bool = False,
        array_size: int = 1,
        channels: list[ReadChannel] | None = None,
        nan_fill_enabled: bool = False,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = ReadConfig(
            device=device,
            sample_rate=Rate(sample_rate),
            stream_rate=Rate(stream_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            array_mode=array_mode,
            array_size=array_size,
            channels=channels if channels is not None else [],
            nan_fill_enabled=nan_fill_enabled,
        )
        task.assign_keys(self.config.channels)

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration."""
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties)
        if "read" not in props:
            props["read"] = {"index": 0, "channels": {}}
        channels = device.ensure_object(props, "read", "channels")
        for ch in self.config.channels:
            if not ch.node_id:
                continue
            if ch.field:
                device.ensure_object(props, "read", "fields", ch.node_id)[ch.field] = (
                    ch.channel
                )
            else:
                channels[ch.node_id] = ch.channel
        dev.properties = props
        return device_client.create(dev)


class WriteTask(
    task.StarterStopperMixin, task.JSONConfigMixin[WriteConfig], task.Protocol
):
    """A write task for sending commands to OPC UA devices. For detailed information on
    configuring and operating an OPC UA write task, see
    https://docs.synnaxlabs.com/reference/hardware/opc-ua/write-task

    :param device: The key of the Synnax OPC UA device to write to.
    :param name: A human-readable name for the task.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The OPC UA nodes to write to, or the methods to call, and the
        Synnax channels to read command values from.
    """

    TYPE = "opc_write"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        auto_start: bool = False,
        channels: list[WriteChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            device=device,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration."""
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties)
        channels = device.ensure_object(props, "write", "channels")
        for ch in self.config.channels:
            # A method can take many channels, so the device binds only nodes.
            if ch.node_id and isinstance(ch, NodeWriteChannel):
                channels[ch.node_id] = ch.cmd_channel
        dev.properties = props
        return device_client.create(dev)


class EventTask(
    task.StarterStopperMixin, task.JSONConfigMixin[EventConfig], task.Protocol
):
    """An event task that records the events of an OPC UA server as JSON samples in a
    Synnax channel. For detailed information on configuring and operating an OPC UA
    event task, see https://docs.synnaxlabs.com/reference/hardware/opc-ua/event-task

    :param device: The key of the Synnax OPC UA device to record events from.
    :param name: A human-readable name for the task.
    :param channel: The key of the JSON channel, with an index, that the task writes
        events to.
    :param notifiers: The node IDs of the objects to record events from. When empty,
        the task records the events of the Server object.
    :param event_types: The node IDs of the event types to record, with their subtypes.
        When empty, the task records events of every type.
    :param fields: The event fields to record beyond the standard fields.
    :param data_saving_disabled: Whether to only stream events for real-time
        consumption instead of saving them permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    """

    TYPE = "opc_event"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        channel: channel.Key = 0,
        notifiers: list[str] | None = None,
        event_types: list[str] | None = None,
        fields: list[EventField] | None = None,
        data_saving_disabled: bool = False,
        auto_start: bool = False,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = EventConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = EventConfig(
            device=device,
            channel=channel,
            notifiers=notifiers if notifiers is not None else [],
            event_types=event_types if event_types is not None else [],
            fields=fields if fields is not None else [],
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
        )


MAKE = "opc"
MODEL = "OPC UA"


class Device(device.Device):
    """An OPC UA server device.

    :param endpoint: The OPC UA server endpoint URL (e.g. "opc.tcp://localhost:4840/").
    :param username: Username for authentication.
    :param password: Password for authentication.
    :param security_mode: Security mode: "None", "Sign", or "SignAndEncrypt".
    :param security_policy: Security policy name.
    :param client_cert: Path to the client certificate on the Driver's computer. Leave
        it and client_private_key empty to use the certificate the Driver generates.
    :param client_private_key: Path to the client private key on the Driver's
        computer.
    :param server_cert: The PEM contents of the server certificate to trust. When
        empty, the Driver trusts the certificate the server presents on its first
        connection.
    :param certificate_validation_disabled: Whether the Driver accepts any server
        certificate. Use only to troubleshoot a connection.
    :param client_timeout_ms: The timeout of each request and connection attempt, in
        milliseconds.
    :param secure_channel_lifetime_ms: The lifetime of each secure channel before the
        Driver renews it, in milliseconds.
    :param session_timeout_ms: How long the server keeps a session with no requests,
        in milliseconds.
    :param name: Human-readable name for the device.
    :param location: Physical location or description.
    :param rack: Rack key this device belongs to.
    :param key: Unique key for the device. Auto-generated if empty.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        endpoint: str,
        username: str = "",
        password: str = "",
        security_mode: SecurityMode = "None",
        security_policy: SecurityPolicy = "None",
        client_cert: str = "",
        client_private_key: str = "",
        server_cert: str = "",
        certificate_validation_disabled: bool = False,
        client_timeout_ms: int = 5_000,
        secure_channel_lifetime_ms: int = 600_000,
        session_timeout_ms: int = 1_200_000,
        name: str = "",
        location: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = True,
    ):
        if not key:
            key = str(uuid4())
        # The Driver expects snake_case property names.
        connection: dict[str, JsonValue] = {
            "endpoint": endpoint,
            "security_mode": security_mode,
            "security_policy": security_policy,
            "certificate_validation_disabled": certificate_validation_disabled,
            "client_timeout_ms": client_timeout_ms,
            "secure_channel_lifetime_ms": secure_channel_lifetime_ms,
            "session_timeout_ms": session_timeout_ms,
        }
        if username:
            connection["username"] = username
        if password:
            connection["password"] = password
        if client_cert:
            connection["client_certificate"] = client_cert
        if client_private_key:
            connection["client_private_key"] = client_private_key
        if server_cert:
            connection["server_certificate"] = server_cert
        props: dict[str, JsonValue] = {
            "version": "2.0.0",
            "connection": connection,
            "read": {"indexes": [], "channels": {}, "fields": {}},
            "write": {"channels": {}},
        }
        super().__init__(
            key=key,
            location=location,
            rack=rack,
            name=name,
            make=MAKE,
            model=MODEL,
            configured=configured,
            properties=props,
        )
