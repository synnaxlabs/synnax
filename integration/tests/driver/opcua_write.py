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

from tests.driver.opcua_task import OPCUAWriteTaskCase


class OPCUAWriteFloat(OPCUAWriteTaskCase):
    task_name = "OPCUA Write Float"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[opcua.WriteChannel]:
        return [
            opcua.WriteChannel(
                cmd_channel=client.channels.create(
                    name="opcua_cmd_0",
                    data_type=sy.DataType.FLOAT32,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=18",
            ),
            opcua.WriteChannel(
                cmd_channel=client.channels.create(
                    name="opcua_cmd_1",
                    data_type=sy.DataType.FLOAT32,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=19",
            ),
            opcua.WriteChannel(
                cmd_channel=client.channels.create(
                    name="opcua_cmd_2",
                    data_type=sy.DataType.FLOAT32,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                node_id="NS=2;I=20",
            ),
        ]
