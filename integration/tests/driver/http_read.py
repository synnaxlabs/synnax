#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import urllib.request

from examples.http_sim import STATS_PATH, HTTPSlowSim

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


class HTTPReadCappedFanOut(HTTPReadTaskCase):
    """Polls more endpoints than the device allows in flight at once.

    The slow sim sleeps on every request, so the cap is visible in the peak in-flight
    count the server reports.
    """

    task_name = "HTTP Read Capped Fan Out"
    sim_classes = [HTTPSlowSim]
    TASK_DURATION = 1 * sy.TimeSpan.SECOND
    # Seven endpoints against a cap of 3 leaves a short last round.
    POINTERS = [
        ("/api/v1/data", "/temperature"),
        ("/api/v1/data", "/pressure"),
        ("/api/v1/metrics", "/sensors/sensor_0"),
        ("/api/v1/metrics", "/sensors/sensor_1"),
        ("/api/v1/status", "/uptime"),
        ("/health/detailed", "/uptime_seconds"),
        ("/api/v1/device", "/temperature"),
    ]

    def run(self) -> None:
        self.test_task_exists()
        self.test_start_and_stop()
        self.test_peak_in_flight()

    def test_peak_in_flight(self) -> None:
        """The server never saw more requests at once than the device allows."""
        self.log("Testing: Peak in-flight requests")
        url = f"http://{HTTPSlowSim.host}:{HTTPSlowSim.port}{STATS_PATH}"
        with urllib.request.urlopen(url, timeout=5) as res:
            peak = json.load(res)["peak_in_flight"]
        cap = HTTPSlowSim.max_concurrent_requests
        self.log(f"Peak in-flight {peak}, cap {cap}")
        if peak != cap:
            self.fail(f"Expected a peak of {cap} requests in flight, got {peak}")

    @staticmethod
    def create_channels(
        client: sy.Synnax,
    ) -> tuple[list[http.ReadEndpoint], list[int]]:
        endpoints = []
        keys = []
        for i, (path, pointer) in enumerate(HTTPReadCappedFanOut.POINTERS):
            idx = create_index(client, f"http_capped_{i}_index")
            key = create_channel(
                client,
                name=f"http_capped_{i}",
                data_type=sy.DataType.FLOAT64,
                index=idx.key,
            )
            keys.append(key)
            endpoints.append(
                http.ReadEndpoint(
                    path=path,
                    method="GET",
                    fields=[
                        http.ReadField(
                            pointer=pointer, channel=key, data_type="float64"
                        )
                    ],
                )
            )
        return endpoints, keys
