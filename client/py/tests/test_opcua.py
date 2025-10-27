#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from synnax.hardware.opcua import Channel, ReadTask, WrappedReadTaskConfig


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
                            "name": "",
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
        WrappedReadTaskConfig(config=input_data)

    def test_create_and_retrieve_task(self, client: sy.Synnax):
        """Test that ReadTask can be created and retrieved from the database."""
        task = ReadTask(
            name="test-task",
            device="some-device-key",
            sample_rate=10,
            stream_rate=5,
            array_mode=False,
            array_size=1,
            channels=[
                Channel(
                    key="k09AWoiyLxN",
                    node_id="NS=2;I=8",
                    channel=1234,
                )
            ],
        )
        createdTask = client.hardware.tasks.create(
            name="test-task",
            type="opc_read",
            config=task.config.model_dump_json(),
        )
        ReadTask(createdTask)
