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


def create_endpoint(
    client: sy.Synnax,
    *,
    method: http.Method,
    path: str,
    bindings: list[tuple[sy.icd.JSONField, int]],
    fields: list[sy.icd.JSONField] | None = None,
) -> http.WriteEndpoint:
    """Create an ICD with one JSON message, named after the request, that holds the
    fields of bindings and fields. Return an endpoint that sends that message.

    :param bindings: Each field that a command channel fills, with the channel.
    :param fields: The other fields of the message. Each sends its default.
    """
    message = sy.icd.Message(
        name=f"{method} {path}",
        payload=sy.icd.JSONPayload(fields=[f for f, _ in bindings] + (fields or [])),
    )
    ic = client.icds.create(name=message.name, messages=[message])
    return http.WriteEndpoint(
        method=method,
        path=path,
        icd=ic.key,
        message=sy.icd.WriteMessage(
            message=message.key,
            bindings=[
                sy.icd.ChannelWriteBinding(field=f.key, channel=ch)
                for f, ch in bindings
            ],
        ),
    )


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
        endpoint = create_endpoint(
            client,
            method="PUT",
            path="/api/v1/setpoint",
            bindings=[(sy.icd.FloatJSONField(name="value", pointer="/value"), cmd_key)],
        )
        return [endpoint], [cmd_key]


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
        endpoint = create_endpoint(
            client,
            method="POST",
            path="/api/v1/control",
            bindings=[(sy.icd.FloatJSONField(name="power", pointer="/power"), cmd_key)],
            fields=[
                sy.icd.StringJSONField(name="mode", pointer="/mode", default="AUTO")
            ],
        )
        return [endpoint], [cmd_key]
