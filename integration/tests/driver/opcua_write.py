#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from examples.opcua import OPCUATLSAuthSim, OPCUATLSSim

from tests.driver.opcua_task import OPCUAWriteTaskCase
from tests.driver.task import create_channel, create_index


class OPCUAWriteFloat(OPCUAWriteTaskCase):
    task_name = "OPCUA Write Float"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.WriteChannel]:
        idx = create_index(client, "opcua_write_cmd_time")
        return [
            sy.opcua.WriteChannel(
                cmd_channel=create_channel(
                    client,
                    name=f"opcua_cmd_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={18 + i}",
            )
            for i in range(3)
        ]


class OPCUATLSWriteFloat(OPCUAWriteTaskCase):
    task_name = "OPCUA TLS Write Float"
    sim_classes = [OPCUATLSSim]

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.WriteChannel]:
        idx = create_index(client, "opcua_encrypted_write_cmd_time")
        return [
            sy.opcua.WriteChannel(
                cmd_channel=create_channel(
                    client,
                    name=f"opcua_encrypted_cmd_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={18 + i}",
            )
            for i in range(3)
        ]


class OPCUATLSAuthWriteFloat(OPCUAWriteTaskCase):
    task_name = "OPCUA TLS Auth Write Float"
    sim_classes = [OPCUATLSAuthSim]

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.WriteChannel]:
        idx = create_index(client, "opcua_enc_userpass_write_cmd_time")
        return [
            sy.opcua.WriteChannel(
                cmd_channel=create_channel(
                    client,
                    name=f"enc_userpass_cmd_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={18 + i}",
            )
            for i in range(3)
        ]
