#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any
from uuid import uuid4

from pydantic import BaseModel

from synnax import device, task
from synnax.http.types_gen import (
    Header,
    QueryParam,
    ReadConfig,
    ReadEndpoint,
    WriteConfig,
    WriteEndpoint,
)
from synnax.telem import CrudeRate, Rate

MAKE = "http"
MODEL = "HTTP server"


class ExpectedResponse(BaseModel):
    """Expected response validation for health checks.

    :param pointer: JSON Pointer into the response body (e.g. "/status").
    :param expected_value_type: Type of the expected value: "string", "number",
        "boolean", or "null".
    :param expected_value: The value the response must match.
    """

    pointer: str
    expected_value_type: str
    expected_value: str | float | int | bool | None


class HealthCheck(BaseModel):
    """Health check configuration for an HTTP device.

    The driver periodically pings each HTTP device using this config. If the request
    fails or the response does not match, the device is marked unhealthy.

    :param method: HTTP method for the health check ("GET" or "POST").
    :param path: URL path to ping (e.g. "/health").
    :param headers: Optional headers to include.
    :param query_params: Optional query parameters.
    :param body: Optional request body (POST only).
    :param response: Optional response validation config.
    """

    method: str = "GET"
    path: str = "/health"
    headers: list[Header] | None = None
    query_params: list[QueryParam] | None = None
    body: str | None = None
    response: ExpectedResponse | None = None


class ReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A read task for polling HTTP endpoints and writing data to Synnax.

    :param device: The key of the HTTP device to read from.
    :param name: A human-readable name for the task.
    :param rate: The polling rate in Hz.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically.
    :param endpoints: The endpoints to poll.
    """

    TYPE = "http_read"
    config: ReadConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        rate: CrudeRate = 1,
        data_saving_disabled: bool = False,
        auto_start: bool = False,
        endpoints: list[ReadEndpoint] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = ReadConfig(
            device=device,
            rate=Rate(rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            endpoints=endpoints if endpoints is not None else [],
        )
        task.assign_keys(self.config.endpoints)
        for ep in self.config.endpoints:
            task.assign_keys(ep.fields)

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
    :param endpoints: The endpoints to send commands to.
    """

    TYPE = "http_write"
    config: WriteConfig
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
            self.config = WriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            device=device,
            auto_start=auto_start,
            endpoints=endpoints if endpoints is not None else [],
        )
        task.assign_keys(self.config.endpoints)
        for ep in self.config.endpoints:
            task.assign_keys(ep.fields)

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
    """An HTTP server device.

    The device location stores the host:port (e.g. "127.0.0.1:8081"), and the
    ``secure`` property determines the URL scheme (https:// vs http://). The Driver
    constructs the full base URL from these at runtime.

    :param host: Host and port of the HTTP server (e.g. "127.0.0.1:8081").
    :param secure: Whether to use HTTPS (True) or HTTP (False).
    :param timeout_ms: Request timeout in milliseconds.
    :param verify_ssl: Whether to verify SSL certificates.
    :param auth: Authentication config dict (see examples below).
    :param health_check: Health check endpoint config.
    :param name: Human-readable name for the device.
    :param rack: Rack key this device belongs to.
    :param key: Unique key. Auto-generated if empty.
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
