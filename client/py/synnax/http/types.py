#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal
from uuid import uuid4

from pydantic import BaseModel

from synnax import channel as channel_
from synnax import device, task

MAKE = "http"
MODEL = "HTTP server"


class ExpectedResponse(BaseModel):
    """Expected response validation for health checks."""

    pointer: str
    """JSON Pointer into the response body (e.g., '/status')."""
    expected_value_type: str
    """Type of the expected value: 'string', 'number', 'boolean', 'null'."""
    expected_value: str | float | int | bool | None


class HealthCheck(BaseModel):
    """Health check configuration for an HTTP device.

    When the driver starts, it periodically pings each HTTP device using this config.
    If the request fails or the response doesn't match, the device is marked unhealthy.

    :param method: HTTP method for the health check ('GET' or 'POST').
    :param path: URL path to ping (e.g., '/health'). Required.
    :param headers: Optional headers to include.
    :param query_params: Optional query parameters.
    :param body: Optional request body (POST only).
    :param validate_response: Whether to validate the response body.
    :param response: Expected response config (required when validate_response is True).
    """

    method: str = "GET"
    path: str = "/health"
    headers: dict[str, str] | None = None
    query_params: dict[str, str] | None = None
    body: str | None = None
    validate_response: bool = False
    response: ExpectedResponse | None = None


class ReadField(BaseModel):
    """Configuration for a single field extracted from an HTTP response."""

    enabled: bool = True
    key: str = ""
    pointer: str
    """JSON Pointer path to extract (e.g., '/temperature')."""
    channel: channel_.Key = 0
    data_type: str = "float64"
    name: str = ""
    timestamp_format: str | None = None
    """Timestamp format: 'iso8601', 'unix_sec', 'unix_ms', 'unix_us', 'unix_ns'."""
    enum_values: dict[str, float] | None = None
    """String-to-number mappings for enum fields."""

    def __init__(self, **data: Any) -> None:
        if "key" not in data or not data["key"]:
            data["key"] = str(uuid4())
        super().__init__(**data)


class ReadEndpoint(BaseModel):
    """Configuration for a single HTTP endpoint to poll for data."""

    key: str = ""
    method: str = "GET"
    """HTTP method: 'GET' or 'POST'."""
    path: str
    """URL path relative to the device base URL (e.g., '/api/v1/data')."""
    headers: dict[str, str] | None = None
    query_params: dict[str, str] | None = None
    body: str | None = None
    """Request body (only for POST)."""
    fields: list[ReadField] = []
    index: str | None = None
    """Key of the timing field for hardware timing, None for software timing."""

    def __init__(self, **data: Any) -> None:
        if "key" not in data or not data["key"]:
            data["key"] = str(uuid4())
        super().__init__(**data)


class ReadTaskConfig(task.BaseConfig):
    """Configuration for an HTTP read (polling) task."""

    device: str
    data_saving: bool = True
    rate: float
    """Polling rate in Hz."""
    endpoints: list[ReadEndpoint]


class WriteTaskConfig(task.BaseConfig):
    """Configuration for an HTTP write task."""

    device: str
    endpoints: list["WriteEndpoint"]


class ChannelField(BaseModel):
    """Configuration for mapping a Synnax channel value into an HTTP request body."""

    pointer: str
    """JSON Pointer where channel value goes in body."""
    json_type: str = "number"
    """JSON type: 'number', 'string', 'boolean'."""
    channel: channel_.Key = 0
    name: str = ""
    data_type: str = "float64"
    time_format: str | None = None


class StaticField(BaseModel):
    """A field with a fixed value in the request body."""

    key: str = ""
    pointer: str
    json_type: str
    type: Literal["static"] = "static"
    value: int | float | str | bool

    def __init__(self, **data: Any) -> None:
        if "key" not in data or not data["key"]:
            data["key"] = str(uuid4())
        super().__init__(**data)


class GeneratedField(BaseModel):
    """A field with an auto-generated value (UUID or timestamp)."""

    key: str = ""
    pointer: str
    type: Literal["generated"] = "generated"
    generator: str
    """Generator type: 'uuid' or 'timestamp'."""
    time_format: str | None = None

    def __init__(self, **data: Any) -> None:
        if "key" not in data or not data["key"]:
            data["key"] = str(uuid4())
        super().__init__(**data)


WriteField = StaticField | GeneratedField


class WriteEndpoint(BaseModel):
    """Configuration for a single HTTP endpoint to send commands to."""

    enabled: bool = True
    key: str = ""
    method: str = "POST"
    """HTTP method: 'POST', 'PUT', 'PATCH'."""
    path: str
    """URL path relative to the device base URL."""
    headers: dict[str, str] | None = None
    query_params: dict[str, str] | None = None
    channel: ChannelField
    """The Synnax channel whose values drive requests to this endpoint."""
    fields: list[WriteField] = []
    """Additional static or generated fields in the request body."""

    def __init__(self, **data: Any) -> None:
        if "key" not in data or not data["key"]:
            data["key"] = str(uuid4())
        super().__init__(**data)


# Resolve forward reference
WriteTaskConfig.model_rebuild()


class ReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A read task for polling HTTP endpoints and writing data to Synnax.

    :param device: The key of the HTTP device to read from.
    :param name: A human-readable name for the task.
    :param rate: The polling rate in Hz.
    :param data_saving: Whether to save data permanently.
    :param auto_start: Whether to start the task automatically.
    :param endpoints: List of endpoint configurations to poll.
    """

    TYPE = "http_read"
    config: ReadTaskConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: str = "",
        name: str = "",
        rate: float = 1,
        data_saving: bool = True,
        auto_start: bool = False,
        endpoints: list[ReadEndpoint] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadTaskConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = ReadTaskConfig(
            device=device,
            rate=rate,
            data_saving=data_saving,
            auto_start=auto_start,
            endpoints=endpoints if endpoints is not None else [],
        )

    def to_payload(self) -> task.Payload:
        pld = self._internal.to_payload()
        pld.config = self.config.model_dump(exclude_none=True)
        return pld

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Sync channel mappings to device properties."""
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties)
        if "read" not in props:
            props["read"] = {}
        for ep in self.config.endpoints:
            channels: dict[str, int] = {}
            index_key: str | None = None
            for field in ep.fields:
                channels[field.pointer] = field.channel
                if ep.index is not None and field.key == ep.index:
                    index_key = field.key
            props["read"][ep.path] = {"index": index_key, "channels": channels}
        dev.properties = props
        return device_client.create(dev)


class WriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A write task for sending commands to HTTP endpoints.

    :param device: The key of the HTTP device to write to.
    :param name: A human-readable name for the task.
    :param auto_start: Whether to start the task automatically.
    :param endpoints: List of endpoint configurations to send commands to.
    """

    TYPE = "http_write"
    config: WriteTaskConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        auto_start: bool = False,
        endpoints: list[WriteEndpoint] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteTaskConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteTaskConfig(
            device=device,
            auto_start=auto_start,
            endpoints=endpoints if endpoints is not None else [],
        )

    def to_payload(self) -> task.Payload:
        pld = self._internal.to_payload()
        pld.config = self.config.model_dump(exclude_none=True)
        return pld

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Sync channel mappings to device properties."""
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties)
        if "write" not in props:
            props["write"] = {}
        for ep in self.config.endpoints:
            props["write"][ep.path] = ep.channel.channel
        dev.properties = props
        return device_client.create(dev)


class Device(device.Device):
    """HTTP server device configuration.

    The device location stores the host:port (e.g., '127.0.0.1:8081'), and the
    ``secure`` property determines the URL scheme (https:// vs http://). The C++ driver
    constructs the full base_url from these at runtime.

    :param host: Host and port of the HTTP server (e.g., '127.0.0.1:8081').
    :param secure: Whether to use HTTPS (True) or HTTP (False).
    :param timeout_ms: Request timeout in milliseconds.
    :param verify_ssl: Whether to verify SSL certificates.
    :param auth: Authentication config dict (see examples below).
    :param health_check: Health check endpoint config dict.
    :param name: Human-readable name for the device.
    :param rack: Rack key this device belongs to.
    :param key: Unique key (auto-generated if empty).
    :param configured: Whether the device has been configured.

    Auth examples::

        # No auth (default)
        {"type": "none"}

        # Bearer token
        {"type": "bearer", "token": "my-token"}

        # Basic auth
        {"type": "basic", "username": "user", "password": "pass"}

        # API key in header
        {"type": "api_key", "send_as": "header", "header": "X-API-Key", "key": "abc"}

        # API key in query param
        {"type": "api_key", "send_as": "query_param", "parameter": "api_key", "key": "abc"}
    """

    def __init__(
        self,
        *,
        host: str,
        secure: bool = True,
        timeout_ms: int = 100,
        verify_ssl: bool = True,
        auth: dict[str, Any] | None = None,
        health_check: HealthCheck | None = None,
        name: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = True,
    ):
        if not key:
            key = str(uuid4())

        if health_check is None:
            health_check = HealthCheck()

        props: dict[str, Any] = {
            "secure": secure,
            "verify_ssl": verify_ssl,
            "timeout_ms": timeout_ms,
            "auth": auth if auth is not None else {"type": "none"},
            "health_check": health_check.model_dump(exclude_none=True),
            "read": {},
            "write": {},
            "version": 1,
        }

        super().__init__(
            key=key,
            location=host,
            rack=rack,
            name=name,
            make=MAKE,
            model=MODEL,
            configured=configured,
            properties=props,
        )
