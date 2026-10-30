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


@pytest.mark.opcua
class TestOPCDevice:
    def test_certificate_properties(self) -> None:
        """Test that Device writes the certificate settings that the Driver reads."""
        dev = sy.opcua.Device(
            endpoint="opc.tcp://localhost:4840",
            security_mode="SignAndEncrypt",
            security_policy="Basic256Sha256",
            server_cert="-----BEGIN CERTIFICATE-----",
            certificate_validation_disabled=True,
        )
        assert dev.properties["version"] == "2.0.0"
        assert dev.properties["connection"] == {
            "endpoint": "opc.tcp://localhost:4840",
            "security_mode": "SignAndEncrypt",
            "security_policy": "Basic256Sha256",
            "certificate_validation_disabled": True,
            "client_timeout_ms": 5_000,
            "secure_channel_lifetime_ms": 600_000,
            "session_timeout_ms": 1_200_000,
            "server_certificate": "-----BEGIN CERTIFICATE-----",
        }

    def test_timeout_properties(self) -> None:
        """Should write the timeouts that the Driver reads."""
        dev = sy.opcua.Device(
            endpoint="opc.tcp://localhost:4840",
            client_timeout_ms=2_000,
            secure_channel_lifetime_ms=60_000,
            session_timeout_ms=120_000,
        )
        connection = dev.properties["connection"]
        assert isinstance(connection, dict)
        assert connection["client_timeout_ms"] == 2_000
        assert connection["secure_channel_lifetime_ms"] == 60_000
        assert connection["session_timeout_ms"] == 120_000


@pytest.mark.opcua
class TestOPCReadTask:
    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_config",
                "data": {
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "sample_rate": 10,
                    "stream_rate": 5,
                    "array_mode": False,
                    "array_size": 1,
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "node_name": "",
                            "key": "k09AWoiyLxN",
                            "node_id": "NS=2;I=8",
                            "channel": 1234,
                            "disabled": False,
                            "is_index": False,
                        },
                    ],
                },
            },
            {
                "name": "non_array_sampling",
                "data": {
                    "device": "some-device-key",
                    "sample_rate": 10,
                    "stream_rate": 5,
                    "array_mode": False,
                    "data_saving_disabled": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "key": "k09AWoiyLxN",
                            "node_id": "NS=2;I=8",
                            "channel": 1234,
                            "disabled": False,
                            "is_index": False,
                        },
                    ],
                },
            },
            {
                "name": "namespace_uri_node_id",
                "data": {
                    "device": "some-device-key",
                    "sample_rate": 10,
                    "stream_rate": 5,
                    "channels": [
                        {
                            "key": "k09AWoiyLxN",
                            "node_id": "nsu=urn:synnax:test;s=Pressure",
                            "channel": 1234,
                        },
                    ],
                },
            },
            {
                "name": "array_sampling",
                "data": {
                    "device": "some-device-key",
                    "sample_rate": 10,
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "array_mode": True,
                    "array_size": 1,
                    "channels": [
                        {
                            "key": "k09AWoiyLxN",
                            "node_id": "NS=2;I=8",
                            "node_name": "some-node-name",
                            "channel": 1234,
                            "disabled": False,
                            "is_index": False,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_opc_read_task(self, test_data: dict[str, object]) -> None:
        """Test that ReadConfig can parse various configurations correctly."""
        input_data = test_data["data"]
        sy.opcua.ReadConfig.model_validate(input_data)

    def test_create_and_retrieve_task(self, client: sy.Synnax) -> None:
        """Test that ReadTask can be created and retrieved from the database."""
        task = sy.opcua.ReadTask(
            name="test-task",
            device="some-device-key",
            sample_rate=10,
            stream_rate=5,
            array_mode=False,
            array_size=1,
            channels=[
                sy.opcua.ReadChannel(
                    key="k09AWoiyLxN",
                    node_id="NS=2;I=8",
                    channel=1234,
                )
            ],
        )
        createdTask = client.tasks.create(
            name="test-task",
            type="opc_read",
            config=task.config,
        )
        sy.opcua.ReadTask(createdTask)

    def test_nan_fill_enabled_round_trip(self, client: sy.Synnax) -> None:
        """Should keep nan_fill_enabled through a create and retrieve."""
        task = sy.opcua.ReadTask(
            name="test-task", device="some-device-key", nan_fill_enabled=True
        )
        created = client.tasks.create(
            name="test-task", type="opc_read", config=task.config
        )
        retrieved = client.tasks.retrieve(key=created.key)
        assert sy.opcua.ReadTask(retrieved).config.nan_fill_enabled is True

    def test_mode_defaults_to_sampled(self) -> None:
        """Should read in sampled mode without backfill by default."""
        task = sy.opcua.ReadTask(name="test-task", device="some-device-key")
        assert task.config.mode == "sampled"
        assert task.config.history_backfill_enabled is False

    def test_on_change_mode_round_trip(self, client: sy.Synnax) -> None:
        """Should keep the On change mode and backfill through a create and
        retrieve."""
        task = sy.opcua.ReadTask(
            name="test-task",
            device="some-device-key",
            mode="on_change",
            history_backfill_enabled=True,
        )
        created = client.tasks.create(
            name="test-task", type="opc_read", config=task.config
        )
        retrieved = sy.opcua.ReadTask(client.tasks.retrieve(key=created.key))
        assert retrieved.config.mode == "on_change"
        assert retrieved.config.history_backfill_enabled is True

    def test_update_device_properties_maps_fields(self, client: sy.Synnax) -> None:
        """Should map a field channel by node and field, apart from node channels."""
        rack = client.racks.retrieve_embedded_rack()
        dev = sy.opcua.Device(
            endpoint="opc.tcp://localhost:4840",
            name="Test OPC UA Device",
            location="opc.tcp://localhost:4840",
            rack=rack.key,
        )
        client.devices.create(dev)
        task = sy.opcua.ReadTask(
            name="test-task",
            device=dev.key,
            channels=[
                sy.opcua.ReadChannel(
                    node_id="NS=1;S=TestMotor", field="Speed", channel=11
                ),
                sy.opcua.ReadChannel(
                    node_id="NS=1;S=TestMotor", field="Position.Y", channel=12
                ),
                sy.opcua.ReadChannel(node_id="NS=1;S=TestDouble", channel=13),
            ],
        )
        task.update_device_properties(client.devices)
        read = client.devices.retrieve(key=dev.key).properties["read"]
        assert isinstance(read, dict)
        assert read["fields"] == {"NS=1;S=TestMotor": {"Speed": 11, "Position.Y": 12}}
        assert read["channels"] == {"NS=1;S=TestDouble": 13}


@pytest.mark.opcua
class TestOPCWriteTask:
    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_write_config",
                "data": {
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "auto_start": True,
                    "channels": [
                        {
                            "key": "k09AWoiyLxN",
                            "type": "node",
                            "node_id": "ns=2;i=8",
                            "cmd_channel": 1234,
                            "disabled": False,
                        },
                    ],
                },
            },
            {
                "name": "multiple_channels_config",
                "data": {
                    "device": "some-device-key",
                    "auto_start": False,
                    "channels": [
                        {
                            "key": "k09AWoiyLxN",
                            "type": "node",
                            "node_id": "ns=2;i=8",
                            "cmd_channel": 1234,
                            "disabled": False,
                        },
                        {
                            "key": "k10BWoiyLxN",
                            "type": "node",
                            "node_id": "ns=2;i=10",
                            "cmd_channel": 5678,
                            "disabled": False,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_opc_write_task(self, test_data: dict[str, object]) -> None:
        """Test that WriteConfig can parse various configurations correctly."""
        input_data = test_data["data"]
        sy.opcua.WriteConfig.model_validate(input_data)

    def test_create_and_retrieve_write_task(self, client: sy.Synnax) -> None:
        """Test that WriteTask can be created and retrieved from the database."""
        task = sy.opcua.WriteTask(
            name="test-write-task",
            device="some-device-key",
            auto_start=True,
            channels=[
                sy.opcua.NodeWriteChannel(
                    key="k09AWoiyLxN",
                    node_id="ns=2;i=8",
                    cmd_channel=1234,
                )
            ],
        )
        createdTask = client.tasks.create(
            name="test-write-task",
            type="opc_write",
            config=task.config,
        )
        sy.opcua.WriteTask(createdTask)

    def test_write_task_disabled_channels(self, client: sy.Synnax) -> None:
        """Test that disabled channels are handled correctly."""
        task = sy.opcua.WriteTask(
            name="test-disabled-channels",
            device="some-device-key",
            auto_start=False,
            channels=[
                sy.opcua.NodeWriteChannel(
                    key="k09AWoiyLxN",
                    node_id="ns=2;i=8",
                    cmd_channel=1234,
                    disabled=False,
                ),
                sy.opcua.NodeWriteChannel(
                    key="k10BWoiyLxN",
                    node_id="ns=2;i=9",
                    cmd_channel=5678,
                    disabled=True,
                ),
            ],
        )
        # Both channels should be in the config (enabled and disabled)
        assert len(task.config.channels) == 2
        assert task.config.channels[0].disabled is False
        assert task.config.channels[1].disabled is True

    def test_write_channel_auto_key_generation(self) -> None:
        """Test that the WriteTask assigns keys to channels missing one."""
        task = sy.opcua.WriteTask(
            name="test",
            device="some-device-key",
            channels=[
                sy.opcua.NodeWriteChannel(
                    node_id="ns=2;i=8",
                    cmd_channel=1234,
                )
            ],
        )
        channel = task.config.channels[0]
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_write_task_serialization_round_trip(self, client: sy.Synnax) -> None:
        """Test that task can be serialized and deserialized correctly."""
        original_task = sy.opcua.WriteTask(
            name="test-round-trip",
            device="some-device-key",
            auto_start=False,
            channels=[
                sy.opcua.NodeWriteChannel(
                    key="k09AWoiyLxN",
                    node_id="ns=2;i=8",
                    cmd_channel=1234,
                    disabled=False,
                ),
                sy.opcua.NodeWriteChannel(
                    key="k10BWoiyLxN",
                    node_id="ns=2;i=10",
                    cmd_channel=5678,
                    disabled=True,
                ),
            ],
        )

        # Create task in database
        created_task = client.tasks.create(
            name="test-round-trip",
            type="opc_write",
            config=original_task.config,
        )

        # Deserialize from database
        retrieved_task = sy.opcua.WriteTask(created_task)

        # Verify all fields match
        assert retrieved_task.config.device == original_task.config.device
        assert retrieved_task.config.auto_start == original_task.config.auto_start
        assert len(retrieved_task.config.channels) == len(original_task.config.channels)

        for orig_ch, retr_ch in zip(
            original_task.config.channels, retrieved_task.config.channels, strict=True
        ):
            assert retr_ch.key == orig_ch.key
            assert retr_ch.node_id == orig_ch.node_id
            assert retr_ch.cmd_channel == orig_ch.cmd_channel
            assert retr_ch.disabled == orig_ch.disabled

    def test_method_channel_round_trip(self, client: sy.Synnax) -> None:
        """Should keep the object and the arguments of a method channel."""
        task = sy.opcua.WriteTask(
            name="test-method",
            device="some-device-key",
            channels=[
                sy.opcua.MethodWriteChannel(
                    node_id="ns=1;s=Pump.Configure",
                    cmd_channel=1234,
                    object_id="ns=1;s=Pump",
                    arguments=[
                        sy.opcua.ValueMethodArgument(value="auto", name="mode"),
                        sy.opcua.CommandMethodArgument(
                            name="speed", data_type=sy.DataType.FLOAT64
                        ),
                    ],
                )
            ],
        )
        created = client.tasks.create(
            name="test-method", type="opc_write", config=task.config
        )
        [ch] = sy.opcua.WriteTask(created).config.channels
        assert isinstance(ch, sy.opcua.MethodWriteChannel)
        assert ch.object_id == "ns=1;s=Pump"
        assert ch.arguments == [
            sy.opcua.ValueMethodArgument(value="auto", name="mode"),
            sy.opcua.CommandMethodArgument(name="speed", data_type=sy.DataType.FLOAT64),
        ]

    def test_update_device_properties_skips_methods(self, client: sy.Synnax) -> None:
        """Should bind node channels to the device and leave method channels out."""
        rack = client.racks.retrieve_embedded_rack()
        dev = sy.opcua.Device(
            endpoint="opc.tcp://localhost:4840",
            name="Test OPC UA Device",
            location="opc.tcp://localhost:4840",
            rack=rack.key,
        )
        client.devices.create(dev)
        task = sy.opcua.WriteTask(
            name="test-task",
            device=dev.key,
            channels=[
                sy.opcua.NodeWriteChannel(node_id="NS=1;S=Valve", cmd_channel=21),
                sy.opcua.MethodWriteChannel(
                    node_id="NS=1;S=Pump.Start",
                    cmd_channel=22,
                    object_id="NS=1;S=Pump",
                ),
            ],
        )
        task.update_device_properties(client.devices)
        write = client.devices.retrieve(key=dev.key).properties["write"]
        assert isinstance(write, dict)
        assert write["channels"] == {"NS=1;S=Valve": 21}


@pytest.mark.opcua
class TestOPCEventTask:
    def test_create_and_retrieve_event_task(self, client: sy.Synnax) -> None:
        """Should keep the sources, event types, and fields of an event task."""
        task = sy.opcua.EventTask(
            name="test-event-task",
            device="some-device-key",
            channel=1234,
            notifiers=["ns=1;s=Pump"],
            event_types=["ns=1;s=TestEventType"],
            fields=[sy.opcua.EventField(name="level", browse_path="1:Level")],
            data_saving_disabled=True,
        )
        created = client.tasks.create(
            name="test-event-task", type=sy.opcua.EventTask.TYPE, config=task.config
        )
        config = sy.opcua.EventTask(created).config
        assert config.channel == 1234
        assert config.notifiers == ["ns=1;s=Pump"]
        assert config.event_types == ["ns=1;s=TestEventType"]
        assert [(f.name, f.browse_path) for f in config.fields] == [
            ("level", "1:Level")
        ]
        assert config.fields[0].key != ""
        assert config.data_saving_disabled is True
