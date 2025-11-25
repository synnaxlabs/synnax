#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import synnax as sy
from synnax.hardware import opcua

from tests.driver.opcua_task import OpcuaTaskCase


class OPCUAReadFloat(OpcuaTaskCase):
    """
    OPC UA read task test for float32 channels.

    Tests scalar float32 channels (NodeIds NS=2;I=8, NS=2;I=9).
    """

    def __init__(self, **kwargs: object) -> None:
        super().__init__(task_name="OPCUA Read Float", **kwargs)

    def create_channels(self, *, device: sy.Device) -> list[opcua.ReadChannel]:
        """Create float32 channels."""
        index_c = self.client.channels.create(
            name="opcua_float_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_float_0",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=8",
                data_type="float32",
            ),
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_float_1",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=9",
                data_type="float32",
            ),
        ]


class OPCUAReadBool(OpcuaTaskCase):
    """
    OPC UA read task test for boolean channels.

    Tests scalar boolean channels (UINT8, NodeIds NS=2;I=13, NS=2;I=14).
    """

    def __init__(self, **kwargs: object) -> None:
        super().__init__(task_name="OPCUA Read Bool", **kwargs)

    def create_channels(self, *, device: sy.Device) -> list[opcua.ReadChannel]:
        """Create boolean channels."""
        index_c = self.client.channels.create(
            name="opcua_bool_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_bool_0",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=13",
                data_type="bool",
            ),
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_bool_1",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=14",
                data_type="bool",
            ),
        ]


class OPCUAReadArray(OpcuaTaskCase):
    """
    OPC UA read task test for array mode.

    Tests array mode float32 channels (NodeIds NS=2;I=2, NS=2;I=3).
    """


    def __init__(
        self, 
        *,
        array_mode: bool = True,
        array_size: int = 5,
        **kwargs: object) -> None:

        super().__init__(
            task_name="OPCUA Read Array",
            array_mode=array_mode,
            array_size=array_size,
            **kwargs,
        )

    def create_channels(self, *, device: sy.Device) -> list[opcua.ReadChannel]:
        """Create array channels."""

        index_c = self.client.channels.create(
            name="opcua_array_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_array_0",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=2",
                data_type="float32",
            ),
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_array_1",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=3",
                data_type="float32",
            ),
        ]


class OPCUAReadMixed(OpcuaTaskCase):
    """
    OPC UA read task test with mixed channel types.

    Tests scalar mode with both float32 and boolean channels.
    """

    def __init__(self, **kwargs: object) -> None:
        super().__init__(task_name="OPCUA Read Mixed", **kwargs)

    def create_channels(self, *, device: sy.Device) -> list[opcua.ReadChannel]:
        """Create mixed channel types."""
        index_c = self.client.channels.create(
            name="opcua_mixed_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_float_0",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=8",
                data_type="float32",
            ),
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_float_1",
                    data_type=sy.DataType.FLOAT32,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=9",
                data_type="float32",
            ),
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_bool_0",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=13",
                data_type="bool",
            ),
            opcua.ReadChannel(
                channel=self.client.channels.create(
                    name="my_bool_1",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=14",
                data_type="bool",
            ),
        ]
