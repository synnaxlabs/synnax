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
                            "use_as_index": False,
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
                            "use_as_index": False,
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
                            "use_as_index": False,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_opc_read_task(self, test_data):
        """Test that ReadConfig can parse various configurations correctly."""
        input_data = test_data["data"]
        sy.opcua.ReadConfig.model_validate(input_data)

    def test_create_and_retrieve_task(self, client: sy.Synnax):
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
                            "node_id": "ns=2;i=8",
                            "cmd_channel": 1234,
                            "disabled": False,
                        },
                        {
                            "key": "k10BWoiyLxN",
                            "node_id": "ns=2;i=10",
                            "cmd_channel": 5678,
                            "disabled": False,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_opc_write_task(self, test_data):
        """Test that WriteConfig can parse various configurations correctly."""
        input_data = test_data["data"]
        sy.opcua.WriteConfig.model_validate(input_data)

    def test_create_and_retrieve_write_task(self, client: sy.Synnax):
        """Test that WriteTask can be created and retrieved from the database."""
        task = sy.opcua.WriteTask(
            name="test-write-task",
            device="some-device-key",
            auto_start=True,
            channels=[
                sy.opcua.WriteChannel(
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

    def test_write_task_disabled_channels(self, client: sy.Synnax):
        """Test that disabled channels are handled correctly."""
        task = sy.opcua.WriteTask(
            name="test-disabled-channels",
            device="some-device-key",
            auto_start=False,
            channels=[
                sy.opcua.WriteChannel(
                    key="k09AWoiyLxN",
                    node_id="ns=2;i=8",
                    cmd_channel=1234,
                    disabled=False,
                ),
                sy.opcua.WriteChannel(
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

    def test_write_channel_auto_key_generation(self):
        """Test that the WriteTask assigns keys to channels missing one."""
        task = sy.opcua.WriteTask(
            name="test",
            device="some-device-key",
            channels=[
                sy.opcua.WriteChannel(
                    node_id="ns=2;i=8",
                    cmd_channel=1234,
                )
            ],
        )
        channel = task.config.channels[0]
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_write_task_serialization_round_trip(self, client: sy.Synnax):
        """Test that task can be serialized and deserialized correctly."""
        original_task = sy.opcua.WriteTask(
            name="test-round-trip",
            device="some-device-key",
            auto_start=False,
            channels=[
                sy.opcua.WriteChannel(
                    key="k09AWoiyLxN",
                    node_id="ns=2;i=8",
                    cmd_channel=1234,
                    disabled=False,
                ),
                sy.opcua.WriteChannel(
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
            original_task.config.channels, retrieved_task.config.channels
        ):
            assert retr_ch.key == orig_ch.key
            assert retr_ch.node_id == orig_ch.node_id
            assert retr_ch.cmd_channel == orig_ch.cmd_channel
            assert retr_ch.disabled == orig_ch.disabled


@pytest.mark.opcua
class TestOPCUADeprecatedNames:
    def test_should_resolve_released_channel_aliases(self):
        """Test that names released clients used still resolve."""
        assert sy.opcua.Channel is sy.opcua.ReadChannel
        assert sy.opcua.WriteTaskConfig is sy.opcua.WriteConfig

    def test_should_forward_the_hardware_module(self):
        """Test that the deprecated synnax.hardware.opcua module still forwards."""
        from synnax.hardware import opcua as hardware_opcua

        assert hardware_opcua.ReadTask is sy.opcua.ReadTask
        assert hardware_opcua.Device is sy.opcua.Device
