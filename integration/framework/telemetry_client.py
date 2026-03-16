#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
from collections.abc import Callable
from enum import Enum

import synnax as sy


class TelemetryClient:
    """Manages background telemetry reporting for the test conductor."""

    def __init__(
        self,
        client: sy.Synnax,
        name: str,
        get_state: Callable[[], Enum],
        get_should_stop: Callable[[], bool],
    ) -> None:
        self._client = client
        self._name = name
        self._get_state = get_state
        self._get_should_stop = get_should_stop
        self._thread: threading.Thread | None = None
        self.tlm: dict[str, int | float | sy.TimeStamp] = {
            f"{name}_time": sy.TimeStamp.now(),
            f"{name}_uptime": 0,
            f"{name}_state": get_state().value,
            f"{name}_test_case_count": 0,
            f"{name}_test_cases_ran": 0,
        }

    def start(self) -> None:
        """Start the telemetry background thread."""
        self._thread = threading.Thread(
            target=self._run, daemon=True, name=f"{self._name}_telemetry"
        )
        self._thread.start()

    def stop(self, timeout: float = 5.0) -> bool:
        """Stop the telemetry thread. Returns True if stopped cleanly."""
        if self._thread is not None and self._thread.is_alive():
            self._thread.join(timeout=timeout)
            return not self._thread.is_alive()
        return True

    def _run(self) -> None:
        loop = sy.Loop(sy.Rate.HZ * 5)

        time_ch = self._client.channels.create(
            name=f"{self._name}_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        uptime_ch = self._client.channels.create(
            name=f"{self._name}_uptime",
            data_type=sy.DataType.UINT32,
            index=time_ch.key,
            retrieve_if_name_exists=True,
        )
        state_ch = self._client.channels.create(
            name=f"{self._name}_state",
            data_type=sy.DataType.UINT8,
            index=time_ch.key,
            retrieve_if_name_exists=True,
        )
        test_case_count_ch = self._client.channels.create(
            name=f"{self._name}_test_case_count",
            data_type=sy.DataType.UINT32,
            index=time_ch.key,
            retrieve_if_name_exists=True,
        )
        test_cases_ran_ch = self._client.channels.create(
            name=f"{self._name}_test_cases_ran",
            data_type=sy.DataType.UINT32,
            index=time_ch.key,
            retrieve_if_name_exists=True,
        )

        start_time = sy.TimeStamp.now()
        self.tlm[f"{self._name}_time"] = start_time

        with self._client.open_writer(
            start=start_time,
            channels=[
                time_ch,
                uptime_ch,
                state_ch,
                test_case_count_ch,
                test_cases_ran_ch,
            ],
            name=self._name,
        ) as writer:
            writer.write(self.tlm)

            while loop.wait() and not self._get_should_stop():
                now = sy.TimeStamp.now()
                uptime_value = (now - start_time) / 1e9

                self.tlm[f"{self._name}_time"] = now
                self.tlm[f"{self._name}_uptime"] = uptime_value
                self.tlm[f"{self._name}_state"] = self._get_state().value
                writer.write(self.tlm)
