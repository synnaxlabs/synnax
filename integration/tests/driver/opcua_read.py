#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from tests.driver.opcua_task import OPCUAReadTaskCase
from tests.driver.task import create_channel, create_index


class OPCUAReadFloat(OPCUAReadTaskCase):
    task_name = "OPCUA Read Float"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_float_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type="float32",
            )
            for i in range(2)
        ]


class OPCUAReadBool(OPCUAReadTaskCase):
    task_name = "OPCUA Read Bool"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_bool_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_bool_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={13 + i}",
                data_type="bool",
            )
            for i in range(2)
        ]


class OPCUAReadArray(OPCUAReadTaskCase):
    task_name = "OPCUA Read Array"
    array_mode = True

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_array_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_array_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={2 + i}",
                data_type="float32",
            )
            for i in range(2)
        ]


class OPCUAReadTimestamp(OPCUAReadTaskCase):
    """Test reading server timestamps as the task index (use_as_index)."""

    task_name = "OPCUA Read Timestamp"
    array_mode = True

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_timestamp_index")
        return [
            sy.opcua.ReadChannel(
                channel=idx.key,
                node_id="NS=2;I=7",
                data_type="datetime",
                use_as_index=True,
            ),
        ] + [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"opcua_ts_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={2 + i}",
                data_type="float32",
            )
            for i in range(2)
        ]

    def test_reconfigure_rate(self) -> None:
        """With server timestamps as index, the array size is controlled by the
        simulator, not the task's sample_rate."""
        self.log("Skipping: test_reconfigure_rate (server-timestamped index)")


class OPCUAReadMixed(OPCUAReadTaskCase):
    task_name = "OPCUA Read Mixed"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "opcua_mixed_index")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type="float32",
            )
            for i in range(2)
        ] + [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"my_bool_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={13 + i}",
                data_type="bool",
            )
            for i in range(2)
        ]
