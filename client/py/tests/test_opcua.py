#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import ValidationError

import synnax as sy


@pytest.mark.opcua
class TestOPCUATask:
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
                    "data_saving": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "node_name": "",
                            "key": "k09AWoiyLxN",
                            "node_id": "NS=2;I=8",
                            "channel": 1234,
                            "enabled": True,
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
                    "data_saving": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "key": "k09AWoiyLxN",
                            "node_id": "NS=2;I=8",
                            "channel": 1234,
                            "enabled": True,
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
                    "data_saving": False,
                    "auto_start": True,
                    "array_mode": True,
                    "array_size": 1,
                    "channels": [
                        {
                            "key": "k09AWoiyLxN",
                            "node_id": "NS=2;I=8",
                            "node_name": "some-node-name",
                            "channel": 1234,
                            "enabled": True,
                            "use_as_index": False,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_opcua_read_task(self, test_data):
        """Test that ReadTaskConfig can parse various configurations correctly."""
        input_data = test_data["data"]
        sy.opcua.WrappedReadTaskConfig(config=input_data)

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
class TestOPCUAWriteTask:
    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_write_config",
                "data": {
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "data_saving": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "key": "k09AWoiyLxN",
                            "node_id": "ns=2;i=8",
                            "cmd_channel": 1234,
                            "enabled": True,
                        },
                    ],
                },
            },
            {
                "name": "multiple_channels_config",
                "data": {
                    "device": "some-device-key",
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "key": "k09AWoiyLxN",
                            "node_id": "ns=2;i=8",
                            "cmd_channel": 1234,
                            "enabled": True,
                        },
                        {
                            "key": "k10BWoiyLxN",
                            "node_id": "ns=2;i=10",
                            "cmd_channel": 5678,
                            "enabled": True,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_opcua_write_task(self, test_data):
        """Test that WriteTaskConfig can parse various configurations correctly."""
        input_data = test_data["data"]
        sy.opcua.WriteTaskConfig.model_validate(input_data)

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

    def test_write_task_empty_channels(self):
        """Test that empty channel list raises validation error."""
        with pytest.raises(ValidationError) as exc_info:
            sy.opcua.WriteTaskConfig(
                device="some-device-key",
                auto_start=True,
                channels=[],
            )
        assert "at least one channel" in str(exc_info.value).lower()

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
                    enabled=True,
                ),
                sy.opcua.WriteChannel(
                    key="k10BWoiyLxN",
                    node_id="ns=2;i=9",
                    cmd_channel=5678,
                    enabled=False,
                ),
            ],
        )
        # Both channels should be in the config (enabled and disabled)
        assert len(task.config.channels) == 2
        assert task.config.channels[0].enabled is True
        assert task.config.channels[1].enabled is False

    def test_write_channel_auto_key_generation(self):
        """Test that WriteChannel auto-generates a key if not provided."""
        channel = sy.opcua.WriteChannel(
            node_id="ns=2;i=8",
            cmd_channel=1234,
        )
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
                    enabled=True,
                ),
                sy.opcua.WriteChannel(
                    key="k10BWoiyLxN",
                    node_id="ns=2;i=10",
                    cmd_channel=5678,
                    enabled=False,
                ),
            ],
        )

        # Serialize to JSON
        config_json = original_task.config

        # Create task in database
        created_task = client.tasks.create(
            name="test-round-trip",
            type="opc_write",
            config=config_json,
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
            assert retr_ch.enabled == orig_ch.enabled


@pytest.mark.opcua
class TestOPCUAReadTaskDeprecation:
    def test_channel_deprecation_warning(self):
        """Test that using Channel emits a deprecation warning."""
        with pytest.warns(DeprecationWarning, match="opcua.Channel is deprecated"):
            sy.opcua.Channel(
                key="test-key",
                node_id="ns=2;i=8",
                channel=1234,
            )
