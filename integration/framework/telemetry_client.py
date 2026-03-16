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

        # Cache channel name strings
        self._ch_time = f"{name}_time"
        self._ch_uptime = f"{name}_uptime"
        self._ch_state = f"{name}_state"
        self._ch_test_case_count = f"{name}_test_case_count"
        self._ch_test_cases_ran = f"{name}_test_cases_ran"

        self.tlm: dict[str, int | float | sy.TimeStamp] = {
            self._ch_time: sy.TimeStamp.now(),
            self._ch_uptime: 0,
            self._ch_state: get_state().value,
            self._ch_test_case_count: 0,
            self._ch_test_cases_ran: 0,
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

    def _create_indexed_channel(
        self, suffix: str, data_type: sy.DataType, index_key: int
    ) -> sy.Channel:
        return self._client.channels.create(
            name=f"{self._name}_{suffix}",
            data_type=data_type,
            index=index_key,
            retrieve_if_name_exists=True,
        )

    def _run(self) -> None:
        loop = sy.Loop(sy.Rate.HZ * 5)

        time_ch = self._client.channels.create(
            name=self._ch_time,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        uptime_ch = self._create_indexed_channel(
            "uptime", sy.DataType.UINT32, time_ch.key
        )
        state_ch = self._create_indexed_channel("state", sy.DataType.UINT8, time_ch.key)
        count_ch = self._create_indexed_channel(
            "test_case_count", sy.DataType.UINT32, time_ch.key
        )
        ran_ch = self._create_indexed_channel(
            "test_cases_ran", sy.DataType.UINT32, time_ch.key
        )

        start_time = sy.TimeStamp.now()
        self.tlm[self._ch_time] = start_time

        with self._client.open_writer(
            start=start_time,
            channels=[time_ch, uptime_ch, state_ch, count_ch, ran_ch],
            name=self._name,
        ) as writer:
            writer.write(self.tlm)

            while loop.wait() and not self._get_should_stop():
                now = sy.TimeStamp.now()
                uptime_value = (now - start_time) / 1e9

                self.tlm[self._ch_time] = now
                self.tlm[self._ch_uptime] = uptime_value
                self.tlm[self._ch_state] = self._get_state().value
                writer.write(self.tlm)
