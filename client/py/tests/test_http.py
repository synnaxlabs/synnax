#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from x.strings import random_name


@pytest.mark.http
class TestHTTPReadTask:
    """Tests for HTTP Read Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_get_endpoint",
                "data": {
                    "device": "http-device-key",
                    "rate": 1.0,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "GET",
                            "path": "/api/v1/data",
                            "fields": [
                                {
                                    "key": "field-1",
                                    "pointer": "/temperature",
                                    "channel": 1234,
                                    "data_type": "float64",
                                    "disabled": False,
                                    "name": "Temperature",
                                },
                            ],
                        },
                    ],
                },
            },
            {
                "name": "post_endpoint_with_body",
                "data": {
                    "device": "http-device-key",
                    "rate": 5.0,
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "endpoints": [
                        {
                            "key": "ep-2",
                            "method": "POST",
                            "path": "/api/v1/query",
                            "body": '{"filter": "active"}',
                            "fields": [
                                {
                                    "key": "field-1",
                                    "pointer": "/count",
                                    "channel": 5678,
                                    "data_type": "int32",
                                    "disabled": False,
                                    "name": "Count",
                                },
                            ],
                        },
                    ],
                },
            },
            {
                "name": "multiple_endpoints_and_fields",
                "data": {
                    "device": "http-device-key",
                    "rate": 2.0,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "GET",
                            "path": "/api/v1/data",
                            "fields": [
                                {
                                    "key": "f-1",
                                    "pointer": "/temperature",
                                    "channel": 1000,
                                    "data_type": "float64",
                                    "disabled": False,
                                },
                                {
                                    "key": "f-2",
                                    "pointer": "/pressure",
                                    "channel": 2000,
                                    "data_type": "float64",
                                    "disabled": False,
                                },
                            ],
                        },
                        {
                            "key": "ep-2",
                            "method": "GET",
                            "path": "/api/v1/metrics",
                            "fields": [
                                {
                                    "key": "f-3",
                                    "pointer": "/sensors/sensor_0",
                                    "channel": 3000,
                                    "data_type": "float64",
                                    "disabled": False,
                                },
                            ],
                        },
                    ],
                },
            },
            {
                "name": "endpoint_with_headers_and_query_params",
                "data": {
                    "device": "http-device-key",
                    "rate": 1.0,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "GET",
                            "path": "/api/v1/data",
                            "headers": [
                                {"name": "Accept", "value": "application/json"}
                            ],
                            "query_params": [{"parameter": "scale", "value": "2.0"}],
                            "fields": [
                                {
                                    "key": "f-1",
                                    "pointer": "/temperature",
                                    "channel": 1234,
                                    "data_type": "float64",
                                    "disabled": False,
                                },
                            ],
                        },
                    ],
                },
            },
            {
                "name": "field_with_time_format",
                "data": {
                    "device": "http-device-key",
                    "rate": 1.0,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "GET",
                            "path": "/api/v1/data",
                            "index": "ts-field",
                            "fields": [
                                {
                                    "key": "ts-field",
                                    "pointer": "/timestamp",
                                    "channel": 100,
                                    "data_type": "timestamp",
                                    "time_format": "unix_sec",
                                    "disabled": False,
                                },
                                {
                                    "key": "val-field",
                                    "pointer": "/value",
                                    "channel": 200,
                                    "data_type": "float64",
                                    "disabled": False,
                                },
                            ],
                        },
                    ],
                },
            },
            {
                "name": "field_with_enum_values",
                "data": {
                    "device": "http-device-key",
                    "rate": 1.0,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "GET",
                            "path": "/api/v1/device",
                            "fields": [
                                {
                                    "key": "f-1",
                                    "pointer": "/power",
                                    "channel": 1234,
                                    "data_type": "float64",
                                    "disabled": False,
                                    "enum_values": [
                                        {"label": "OFF", "value": 0},
                                        {"label": "ON", "value": 1},
                                    ],
                                },
                            ],
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_http_read_task(self, test_data):
        """Test that ReadConfig can parse various endpoint configurations."""
        sy.http.ReadConfig.model_validate(test_data["data"])

    def test_read_task_auto_key_generation(self):
        """Test that the ReadTask assigns keys to endpoints and fields missing one."""
        task = sy.http.ReadTask(
            name="test",
            device="dev-key",
            endpoints=[
                sy.http.ReadEndpoint(
                    path="/api/v1/data",
                    fields=[sy.http.ReadField(pointer="/value", channel=1)],
                ),
            ],
        )
        ep = task.config.endpoints[0]
        assert ep.key != ""
        assert len(ep.key) > 0
        assert ep.fields[0].key != ""
        assert len(ep.fields[0].key) > 0

    def test_read_field_defaults(self):
        """Test that ReadField has correct defaults."""
        field = sy.http.ReadField(pointer="/value", channel=1234)
        assert field.disabled is False
        assert field.data_type == "float64"
        assert field.name == ""
        assert field.time_format is None
        assert field.enum_values == []

    def test_read_endpoint_defaults(self):
        """Test that ReadEndpoint has correct defaults."""
        ep = sy.http.ReadEndpoint(path="/data", fields=[])
        assert ep.method == "GET"
        assert ep.headers == []
        assert ep.query_params == []
        assert ep.body == ""
        assert ep.index == ""

    def test_read_config_defaults(self):
        """Test that ReadConfig has correct defaults."""
        config = sy.http.ReadConfig(
            device="dev-key",
            rate=sy.Rate(1),
            endpoints=[],
        )
        assert config.data_saving_disabled is False
        assert config.auto_start is False

    def test_read_task_none_excluded_from_serialization(self):
        """Test that None values are excluded from serialized config."""
        task = sy.http.ReadTask(
            name="test",
            device="dev-key",
            rate=1.0,
            endpoints=[
                sy.http.ReadEndpoint(
                    path="/data",
                    fields=[sy.http.ReadField(pointer="/value", channel=1)],
                ),
            ],
        )
        payload = task.to_payload()
        ep = payload.config["endpoints"][0]
        assert ep["body"] == ""
        assert ep["headers"] == []
        assert ep["query_params"] == []
        field = ep["fields"][0]
        assert "timestamp_format" not in field
        assert field["enum_values"] == []

    def test_create_and_retrieve_read_task(self, client: sy.Synnax):
        """Test that ReadTask can be created and retrieved from the database."""
        task = sy.http.ReadTask(
            name="test-http-read-task",
            device="some-device-key",
            rate=2.0,
            endpoints=[
                sy.http.ReadEndpoint(
                    key="ep-1",
                    method="GET",
                    path="/api/v1/data",
                    fields=[
                        sy.http.ReadField(
                            key="f-1",
                            pointer="/temperature",
                            channel=1234,
                            data_type="float64",
                        ),
                    ],
                ),
            ],
        )
        created = client.tasks.create(
            name="test-http-read-task",
            type="http_read",
            config=task.config.model_dump(exclude_none=True),
        )
        sy.http.ReadTask(created)


@pytest.mark.http
class TestHTTPWriteTask:
    """Tests for HTTP Write Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_post_endpoint",
                "data": {
                    "device": "http-device-key",
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "POST",
                            "path": "/api/v1/control",
                            "channel": {
                                "pointer": "/power",
                                "json_type": "string",
                                "channel": 1234,
                                "name": "Power",
                                "data_type": "string",
                            },
                        },
                    ],
                },
            },
            {
                "name": "put_endpoint_with_number",
                "data": {
                    "device": "http-device-key",
                    "auto_start": True,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "PUT",
                            "path": "/api/v1/setpoint",
                            "channel": {
                                "pointer": "/value",
                                "json_type": "number",
                                "channel": 5678,
                                "name": "Setpoint",
                                "data_type": "float64",
                            },
                        },
                    ],
                },
            },
            {
                "name": "patch_endpoint",
                "data": {
                    "device": "http-device-key",
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "PATCH",
                            "path": "/api/v1/config",
                            "channel": {
                                "pointer": "/mode",
                                "json_type": "string",
                                "channel": 9012,
                                "name": "Mode",
                                "data_type": "string",
                            },
                        },
                    ],
                },
            },
            {
                "name": "endpoint_with_static_fields",
                "data": {
                    "device": "http-device-key",
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "POST",
                            "path": "/api/v1/control",
                            "channel": {
                                "pointer": "/value",
                                "json_type": "number",
                                "channel": 1234,
                            },
                            "fields": [
                                {
                                    "key": "sf-1",
                                    "pointer": "/source",
                                    "json_type": "string",
                                    "type": "static",
                                    "value": {"source": "python-client"},
                                },
                                {
                                    "key": "sf-2",
                                    "pointer": "/priority",
                                    "json_type": "number",
                                    "type": "static",
                                    "value": {"priority": 1},
                                },
                            ],
                        },
                    ],
                },
            },
            {
                "name": "endpoint_with_generated_fields",
                "data": {
                    "device": "http-device-key",
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "POST",
                            "path": "/api/v1/events",
                            "channel": {
                                "pointer": "/value",
                                "json_type": "number",
                                "channel": 1234,
                            },
                            "fields": [
                                {
                                    "key": "gf-1",
                                    "pointer": "/id",
                                    "type": "generated",
                                    "generator": "uuid",
                                },
                                {
                                    "key": "gf-2",
                                    "pointer": "/timestamp",
                                    "type": "generated",
                                    "generator": "timestamp",
                                    "time_format": "iso8601",
                                },
                            ],
                        },
                    ],
                },
            },
            {
                "name": "multiple_endpoints",
                "data": {
                    "device": "http-device-key",
                    "auto_start": False,
                    "endpoints": [
                        {
                            "key": "ep-1",
                            "method": "PUT",
                            "path": "/api/v1/setpoint",
                            "channel": {
                                "pointer": "/value",
                                "json_type": "number",
                                "channel": 1000,
                            },
                        },
                        {
                            "key": "ep-2",
                            "method": "POST",
                            "path": "/api/v1/control",
                            "channel": {
                                "pointer": "/power",
                                "json_type": "string",
                                "channel": 2000,
                            },
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_http_write_task(self, test_data):
        """Test that WriteConfig can parse various endpoint configurations."""
        sy.http.WriteConfig.model_validate(test_data["data"])

    def test_write_endpoint_auto_key_generation(self):
        """Test that the WriteTask assigns keys to endpoints and fields missing one."""
        task = sy.http.WriteTask(
            name="test",
            device="dev-key",
            endpoints=[
                sy.http.WriteEndpoint(
                    path="/api/v1/control",
                    channel=sy.http.ChannelField(pointer="/value", channel=1),
                    fields=[
                        sy.http.GeneratedWriteField(
                            type="generated",
                            pointer="/id",
                            generator="uuid",
                        ),
                    ],
                ),
            ],
        )
        ep = task.config.endpoints[0]
        assert ep.key != ""
        assert len(ep.key) > 0
        assert ep.fields[0].key != ""
        assert len(ep.fields[0].key) > 0

    def test_write_endpoint_defaults(self):
        """Test that WriteEndpoint has correct defaults."""
        ep = sy.http.WriteEndpoint(
            path="/control",
            channel=sy.http.ChannelField(pointer="/value", channel=1),
        )
        assert ep.disabled is False
        assert ep.method == "POST"
        assert ep.headers == []
        assert ep.query_params == []
        assert ep.fields == []

    def test_channel_field_defaults(self):
        """Test that ChannelField has correct defaults."""
        cf = sy.http.ChannelField(pointer="/value", channel=1234)
        assert cf.json_type == "number"
        assert cf.data_type == "float64"
        assert cf.name == ""
        assert cf.time_format is None

    def test_static_field_type(self):
        """Test that StaticWriteField carries the static discriminator."""
        sf = sy.http.StaticWriteField(
            type="static",
            pointer="/source",
            json_type="string",
            value={"source": "test"},
        )
        assert sf.type == "static"
        assert sf.value == {"source": "test"}

    def test_generated_field_type(self):
        """Test that GeneratedWriteField carries the generated discriminator."""
        gf = sy.http.GeneratedWriteField(
            type="generated", pointer="/id", generator="uuid"
        )
        assert gf.type == "generated"
        assert gf.generator == "uuid"

    def test_write_task_none_excluded_from_serialization(self):
        """Test that None values are excluded from serialized config."""
        task = sy.http.WriteTask(
            name="test",
            device="dev-key",
            endpoints=[
                sy.http.WriteEndpoint(
                    path="/control",
                    channel=sy.http.ChannelField(pointer="/value", channel=1),
                ),
            ],
        )
        payload = task.to_payload()
        ep = payload.config["endpoints"][0]
        assert ep["headers"] == []
        assert ep["query_params"] == []
        assert "time_format" not in ep["channel"]

    def test_create_and_retrieve_write_task(self, client: sy.Synnax):
        """Test that WriteTask can be created and retrieved from the database."""
        task = sy.http.WriteTask(
            name="test-http-write-task",
            device="some-device-key",
            endpoints=[
                sy.http.WriteEndpoint(
                    key="ep-1",
                    method="PUT",
                    path="/api/v1/setpoint",
                    channel=sy.http.ChannelField(
                        pointer="/value",
                        json_type="number",
                        channel=5678,
                    ),
                ),
            ],
        )
        created = client.tasks.create(
            name="test-http-write-task",
            type="http_write",
            config=task.config.model_dump(exclude_none=True),
        )
        sy.http.WriteTask(created)

    def test_write_task_serialization_round_trip(self, client: sy.Synnax):
        """Test that task can be serialized and deserialized correctly."""
        original = sy.http.WriteTask(
            name="test-round-trip",
            device="some-device-key",
            endpoints=[
                sy.http.WriteEndpoint(
                    key="ep-1",
                    method="PUT",
                    path="/api/v1/setpoint",
                    channel=sy.http.ChannelField(
                        pointer="/value",
                        json_type="number",
                        channel=5678,
                        name="Setpoint",
                    ),
                    fields=[
                        sy.http.StaticWriteField(
                            type="static",
                            key="sf-1",
                            pointer="/source",
                            json_type="string",
                            value={"source": "python"},
                        ),
                    ],
                ),
            ],
        )
        created = client.tasks.create(
            name="test-round-trip",
            type="http_write",
            config=original.config.model_dump(exclude_none=True),
        )
        retrieved = sy.http.WriteTask(created)
        assert retrieved.config.device == original.config.device
        assert len(retrieved.config.endpoints) == len(original.config.endpoints)
        orig_ep = original.config.endpoints[0]
        retr_ep = retrieved.config.endpoints[0]
        assert retr_ep.key == orig_ep.key
        assert retr_ep.method == orig_ep.method
        assert retr_ep.path == orig_ep.path
        assert retr_ep.channel.pointer == orig_ep.channel.pointer
        assert retr_ep.channel.channel == orig_ep.channel.channel


@pytest.mark.http
class TestHTTPDevice:
    """Tests for HTTP Device configuration."""

    def test_device_defaults(self):
        """Test that Device has correct defaults."""
        dev = sy.http.Device(host="127.0.0.1:8080")
        assert dev.location == "127.0.0.1:8080"
        assert dev.make == "http"
        assert dev.model == "HTTP server"
        assert dev.configured is True
        assert dev.key != ""
        props = dev.properties
        assert props["secure"] is True
        assert props["verify_ssl"] is True
        assert props["timeout_ms"] == 100
        assert props["auth"] == {"type": "none"}
        assert props["version"] == 1

    def test_device_custom_config(self):
        """Test that Device accepts custom configuration."""
        dev = sy.http.Device(
            host="192.168.1.100:9090",
            secure=False,
            timeout_ms=5000,
            verify_ssl=False,
            auth={"type": "bearer", "token": "my-token"},
            name="My Server",
        )
        assert dev.location == "192.168.1.100:9090"
        assert dev.name == "My Server"
        props = dev.properties
        assert props["secure"] is False
        assert props["timeout_ms"] == 5000
        assert props["verify_ssl"] is False
        assert props["auth"]["type"] == "bearer"
        assert props["auth"]["token"] == "my-token"

    def test_device_basic_auth(self):
        """Test device with basic authentication."""
        dev = sy.http.Device(
            host="localhost:8080",
            auth={"type": "basic", "username": "admin", "password": "secret"},
        )
        auth = dev.properties["auth"]
        assert auth["type"] == "basic"
        assert auth["username"] == "admin"
        assert auth["password"] == "secret"

    def test_device_api_key_header_auth(self):
        """Test device with API key in header."""
        dev = sy.http.Device(
            host="localhost:8080",
            auth={
                "type": "api_key",
                "send_as": "header",
                "header": "X-API-Key",
                "key": "abc123",
            },
        )
        auth = dev.properties["auth"]
        assert auth["type"] == "api_key"
        assert auth["send_as"] == "header"
        assert auth["header"] == "X-API-Key"
        assert auth["key"] == "abc123"

    def test_device_api_key_query_param_auth(self):
        """Test device with API key in query parameter."""
        dev = sy.http.Device(
            host="localhost:8080",
            auth={
                "type": "api_key",
                "send_as": "query_param",
                "parameter": "api_key",
                "key": "abc123",
            },
        )
        auth = dev.properties["auth"]
        assert auth["type"] == "api_key"
        assert auth["send_as"] == "query_param"
        assert auth["parameter"] == "api_key"

    def test_device_auto_key_generation(self):
        """Test that Device auto-generates a key if not provided."""
        dev1 = sy.http.Device(host="localhost:8080")
        dev2 = sy.http.Device(host="localhost:8080")
        assert dev1.key != ""
        assert dev2.key != ""
        assert dev1.key != dev2.key


@pytest.mark.http
class TestHTTPHealthCheck:
    """Tests for HTTP HealthCheck configuration."""

    def test_health_check_defaults(self):
        """Test that HealthCheck has correct defaults."""
        hc = sy.http.HealthCheck()
        assert hc.method == "GET"
        assert hc.path == "/health"
        assert hc.response is None
        assert hc.headers is None
        assert hc.query_params is None
        assert hc.body is None

    def test_health_check_custom_path(self):
        """Test health check with custom path."""
        hc = sy.http.HealthCheck(path="/api/v1/status")
        assert hc.path == "/api/v1/status"

    def test_health_check_post_with_body(self):
        """Test health check with POST method and body."""
        hc = sy.http.HealthCheck(
            method="POST",
            path="/api/v1/ping",
            body='{"check": true}',
        )
        assert hc.method == "POST"
        assert hc.body == '{"check": true}'

    def test_health_check_with_response_validation(self):
        """Test health check with response validation."""
        hc = sy.http.HealthCheck(
            path="/health",
            response=sy.http.ExpectedResponse(
                pointer="/status",
                expected_value_type="string",
                expected_value="ok",
            ),
        )
        assert hc.response is not None
        assert hc.response.pointer == "/status"
        assert hc.response.expected_value == "ok"

    def test_health_check_none_excluded_from_device_properties(self):
        """Test that None values are excluded from health check in device properties."""
        dev = sy.http.Device(host="localhost:8080")
        hc = dev.properties["health_check"]
        assert "headers" not in hc
        assert "query_params" not in hc
        assert "body" not in hc
        assert "response" not in hc

    def test_health_check_in_device(self):
        """Test that health check is properly stored in device properties."""
        dev = sy.http.Device(
            host="localhost:8080",
            health_check=sy.http.HealthCheck(
                path="/api/status",
                headers=[sy.http.Header(name="Accept", value="application/json")],
            ),
        )
        hc = dev.properties["health_check"]
        assert hc["path"] == "/api/status"
        assert hc["method"] == "GET"
        assert hc["headers"] == [{"name": "Accept", "value": "application/json"}]


@pytest.mark.http
class TestHTTPDevicePropertyUpdates:
    """Tests that device properties are correctly updated with channel mappings."""

    def test_read_task_updates_device_properties(self, client: sy.Synnax):
        """Test that configuring a ReadTask updates device properties."""
        rack = client.racks.retrieve_embedded_rack()
        device = sy.http.Device(
            host="127.0.0.1:8080",
            secure=False,
            name="Test HTTP Read Device",
            rack=rack.key,
        )
        client.devices.create(device)

        suffix = random_name()
        time_ch = client.channels.create(
            name=f"http_time_{suffix}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        temp_ch = client.channels.create(
            name=f"http_temp_{suffix}",
            data_type=sy.DataType.FLOAT64,
            index=time_ch.key,
        )
        pres_ch = client.channels.create(
            name=f"http_pres_{suffix}",
            data_type=sy.DataType.FLOAT64,
            index=time_ch.key,
        )

        task = sy.http.ReadTask(
            name="Test HTTP Read",
            device=device.key,
            rate=1.0,
            endpoints=[
                sy.http.ReadEndpoint(
                    path="/api/v1/data",
                    fields=[
                        sy.http.ReadField(
                            pointer="/temperature",
                            channel=temp_ch.key,
                        ),
                        sy.http.ReadField(
                            pointer="/pressure",
                            channel=pres_ch.key,
                        ),
                    ],
                ),
            ],
        )
        task.update_device_properties(client.devices)

        updated = client.devices.retrieve(key=device.key)
        props = updated.properties
        assert "read" in props
        assert "/api/v1/data" in props["read"]
        ep_props = props["read"]["/api/v1/data"]
        assert ep_props["channels"]["/temperature"] == temp_ch.key
        assert ep_props["channels"]["/pressure"] == pres_ch.key

    def test_write_task_updates_device_properties(self, client: sy.Synnax):
        """Test that configuring a WriteTask updates device properties."""
        rack = client.racks.retrieve_embedded_rack()
        device = sy.http.Device(
            host="127.0.0.1:8080",
            secure=False,
            name="Test HTTP Write Device",
            rack=rack.key,
        )
        client.devices.create(device)

        suffix = random_name()
        cmd_time = client.channels.create(
            name=f"http_cmd_time_{suffix}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        cmd_ch = client.channels.create(
            name=f"http_cmd_{suffix}",
            data_type=sy.DataType.FLOAT64,
            index=cmd_time.key,
        )

        task = sy.http.WriteTask(
            name="Test HTTP Write",
            device=device.key,
            endpoints=[
                sy.http.WriteEndpoint(
                    method="PUT",
                    path="/api/v1/setpoint",
                    channel=sy.http.ChannelField(
                        pointer="/value",
                        channel=cmd_ch.key,
                    ),
                ),
            ],
        )
        task.update_device_properties(client.devices)

        updated = client.devices.retrieve(key=device.key)
        props = updated.properties
        assert "write" in props
        assert "/api/v1/setpoint" in props["write"]
        assert props["write"]["/api/v1/setpoint"] == cmd_ch.key
