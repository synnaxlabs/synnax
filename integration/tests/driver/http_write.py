#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax import http
from tests.driver.http_task import HTTPWriteTaskCase
from tests.driver.task import create_channel, create_index


class HTTPWriteSetpoint(HTTPWriteTaskCase):
    task_name = "HTTP Write Setpoint"

    @staticmethod
    def create_channels(
        client: sy.Synnax,
    ) -> tuple[list[http.WriteEndpoint], list[int]]:
        idx = create_index(client, "http_setpoint_cmd_time")
        cmd_key = create_channel(
            client,
            name="http_setpoint_cmd",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        endpoints = [
            http.WriteEndpoint(
                method="PUT",
                path="/api/v1/setpoint",
                channel=http.ChannelField(
                    pointer="/value",
                    json_type="number",
                    channel=cmd_key,
                    data_type="float64",
                ),
            ),
        ]
        return endpoints, [cmd_key]


class HTTPWriteControl(HTTPWriteTaskCase):
    task_name = "HTTP Write Control"

    @staticmethod
    def create_channels(
        client: sy.Synnax,
    ) -> tuple[list[http.WriteEndpoint], list[int]]:
        idx = create_index(client, "http_control_cmd_time")
        cmd_key = create_channel(
            client,
            name="http_power_cmd",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        endpoints = [
            http.WriteEndpoint(
                method="POST",
                path="/api/v1/control",
                channel=http.ChannelField(
                    pointer="/power",
                    json_type="number",
                    channel=cmd_key,
                    data_type="float64",
                ),
                fields=[
                    http.StaticField(
                        pointer="/mode",
                        json_type="string",
                        value="AUTO",
                    ),
                ],
            ),
        ]
        return endpoints, [cmd_key]
