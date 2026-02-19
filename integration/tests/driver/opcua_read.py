#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax import opcua

from tests.driver.opcua_task import OPCUAReadTaskCase


class OPCUAReadFloat(OPCUAReadTaskCase):
    task_name = "OPCUA Read Float"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[opcua.ReadChannel]:
        index_c = client.channels.create(
            name="opcua_float_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_float_0",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=8",
                data_type="float32",
            ),
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_float_1",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=9",
                data_type="float32",
            ),
        ]


class OPCUAReadBool(OPCUAReadTaskCase):
    task_name = "OPCUA Read Bool"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[opcua.ReadChannel]:
        index_c = client.channels.create(
            name="opcua_bool_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_bool_0",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=13",
                data_type="bool",
            ),
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_bool_1",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=14",
                data_type="bool",
            ),
        ]


class OPCUAReadArray(OPCUAReadTaskCase):
    task_name = "OPCUA Read Array"
    array_mode = True

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[opcua.ReadChannel]:
        index_c = client.channels.create(
            name="opcua_array_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_array_0",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=2",
                data_type="float32",
            ),
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_array_1",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=3",
                data_type="float32",
            ),
        ]


class OPCUAReadTimestamp(OPCUAReadTaskCase):
    """Test reading server timestamps as the task index (use_as_index)."""

    task_name = "OPCUA Read Timestamp"
    array_mode = True

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[opcua.ReadChannel]:
        index_c = client.channels.create(
            name="opcua_timestamp_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            opcua.ReadChannel(
                channel=index_c.key,
                node_id="NS=2;I=7",
                data_type="datetime",
                use_as_index=True,
            ),
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="opcua_ts_float_0",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=2",
                data_type="float32",
            ),
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="opcua_ts_float_1",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=3",
                data_type="float32",
            ),
        ]

    def test_reconfigure_rate(self) -> None:
        """Skipped: With server timestamps as index, the array size is controlled by the
        simulator, not the task's sample_rate."""
        pass


class OPCUAReadMixed(OPCUAReadTaskCase):
    task_name = "OPCUA Read Mixed"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[opcua.ReadChannel]:
        index_c = client.channels.create(
            name="opcua_mixed_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_float_0",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=8",
                data_type="float32",
            ),
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_float_1",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=9",
                data_type="float32",
            ),
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_bool_0",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=13",
                data_type="bool",
            ),
            opcua.ReadChannel(
                channel=client.channels.create(
                    name="my_bool_1",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=14",
                data_type="bool",
            ),
        ]
