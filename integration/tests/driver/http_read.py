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

from tests.driver.http_task import HTTPReadTaskCase
from tests.driver.task import create_channel, create_index


class HTTPReadFloat(HTTPReadTaskCase):
    task_name = "HTTP Read Float"

    @staticmethod
    def create_channels(
        client: sy.Synnax,
    ) -> tuple[list[http.ReadEndpoint], list[int]]:
        idx = create_index(client, "http_float_index")
        temp_key = create_channel(
            client,
            name="http_temperature",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        pressure_key = create_channel(
            client,
            name="http_pressure",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        endpoints = [
            http.ReadEndpoint(
                path="/api/v1/data",
                method="GET",
                fields=[
                    http.ReadField(
                        pointer="/temperature",
                        channel=temp_key,
                        data_type="float64",
                    ),
                    http.ReadField(
                        pointer="/pressure",
                        channel=pressure_key,
                        data_type="float64",
                    ),
                ],
            ),
        ]
        return endpoints, [temp_key, pressure_key]


class HTTPReadMultipleEndpoints(HTTPReadTaskCase):
    task_name = "HTTP Read Multiple Endpoints"

    @staticmethod
    def create_channels(
        client: sy.Synnax,
    ) -> tuple[list[http.ReadEndpoint], list[int]]:
        idx1 = create_index(client, "http_data_ep_index")
        idx2 = create_index(client, "http_metrics_ep_index")
        temp_key = create_channel(
            client,
            name="http_data_temperature",
            data_type=sy.DataType.FLOAT64,
            index=idx1.key,
        )
        sensor_key = create_channel(
            client,
            name="http_metrics_sensor_0",
            data_type=sy.DataType.FLOAT64,
            index=idx2.key,
        )
        endpoints = [
            http.ReadEndpoint(
                path="/api/v1/data",
                method="GET",
                fields=[
                    http.ReadField(
                        pointer="/temperature",
                        channel=temp_key,
                        data_type="float64",
                    ),
                ],
            ),
            http.ReadEndpoint(
                path="/api/v1/metrics",
                method="GET",
                fields=[
                    http.ReadField(
                        pointer="/sensors/sensor_0",
                        channel=sensor_key,
                        data_type="float64",
                    ),
                ],
            ),
        ]
        return endpoints, [temp_key, sensor_key]
