#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
import traceback
from collections.abc import Callable
from enum import Enum
from typing import Any

import synnax as sy
from x import suppress_websocket_errors


class TelemetryWriter:
    """Background thread that writes a telemetry dict at a fixed rate."""

    def __init__(
        self,
        client: sy.Synnax,
        tlm: dict[str, Any],
        channels: list,
        update: Callable[[dict, sy.TimeStamp, float], None],
        should_stop: Callable[[], bool],
        *,
        rate: sy.CrudeTimeSpan = 200 * sy.TimeSpan.MILLISECOND,
        name: str | None = None,
        on_error: Callable[[Exception], None] | None = None,
    ) -> None:
        self._client = client
        self._tlm = tlm
        self._channels = channels
        self._update = update
        self._should_stop = should_stop
        self._rate = rate
        self._name = name
        self._on_error = on_error
        self._thread: threading.Thread | None = None

    def start(self) -> None:
        thread_name = f"{self._name}_telemetry" if self._name else None
        self._thread = threading.Thread(
            target=self._loop, daemon=True, name=thread_name
        )
        self._thread.start()

    def stop(self, timeout: float = 5.0) -> bool:
        if self._thread is not None and self._thread.is_alive():
            self._thread.join(timeout=timeout)
            return not self._thread.is_alive()
        return True

    def is_alive(self) -> bool:
        return self._thread is not None and self._thread.is_alive()

    def _loop(self) -> None:
        loop = sy.Loop(self._rate)
        start_time = sy.TimeStamp.now()
        writer = None

        try:
            open_kwargs: dict[str, Any] = {
                "start": start_time,
                "channels": self._channels,
            }
            if self._name is not None:
                open_kwargs["name"] = self._name
            writer = self._client.open_writer(**open_kwargs)

            self._update(self._tlm, start_time, 0)
            writer.write(self._tlm)

            while loop.wait() and not self._should_stop():
                now = sy.TimeStamp.now()
                uptime = (now - start_time) / sy.TimeSpan.SECOND
                self._update(self._tlm, now, uptime)
                try:
                    writer.write(self._tlm)
                except Exception as e:
                    if self._on_error is not None:
                        self._on_error(e)
                    else:
                        raise

        except Exception:
            if self._on_error is not None:
                traceback.print_exc()
            else:
                raise

        finally:
            if writer is not None:
                with suppress_websocket_errors():
                    now = sy.TimeStamp.now()
                    uptime = (now - start_time) / sy.TimeSpan.SECOND
                    self._update(self._tlm, now, uptime)
                    writer.write(self._tlm)
                    writer.close()


class TelemetryClient:
    """Manages background telemetry reporting for the test conductor."""

    def __init__(
        self,
        client: sy.Synnax,
        name: str,
        get_state: Callable[[], Enum],
        get_should_stop: Callable[[], bool],
    ) -> None:
        self._get_state = get_state

        self._ch_time = f"{name}_time"
        self._ch_uptime = f"{name}_uptime"
        self._ch_state = f"{name}_state"
        self._ch_test_case_count = f"{name}_test_case_count"
        self._ch_test_cases_ran = f"{name}_test_cases_ran"

        time_ch = client.channels.create(
            name=self._ch_time,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )

        def create_indexed(suffix: str, data_type: sy.DataType) -> sy.Channel:
            return client.channels.create(
                name=f"{name}_{suffix}",
                data_type=data_type,
                index=time_ch.key,
                retrieve_if_name_exists=True,
            )

        uptime_ch = create_indexed("uptime", sy.DataType.UINT32)
        state_ch = create_indexed("state", sy.DataType.UINT8)
        count_ch = create_indexed("test_case_count", sy.DataType.UINT32)
        ran_ch = create_indexed("test_cases_ran", sy.DataType.UINT32)

        self.tlm: dict[str, int | float | sy.TimeStamp] = {
            self._ch_time: sy.TimeStamp.now(),
            self._ch_uptime: 0,
            self._ch_state: get_state().value,
            self._ch_test_case_count: 0,
            self._ch_test_cases_ran: 0,
        }

        self._writer = TelemetryWriter(
            client=client,
            tlm=self.tlm,
            channels=[time_ch, uptime_ch, state_ch, count_ch, ran_ch],
            update=self._update_tlm,
            should_stop=get_should_stop,
            rate=sy.Rate.HZ * 5,
            name=name,
        )

    def _update_tlm(
        self, tlm: dict, now: sy.TimeStamp, uptime: float
    ) -> None:
        tlm[self._ch_time] = now
        tlm[self._ch_uptime] = uptime
        tlm[self._ch_state] = self._get_state().value

    def set_test_case_count(self, count: int) -> None:
        self.tlm[self._ch_test_case_count] = count

    def increment_tests_ran(self) -> None:
        self.tlm[self._ch_test_cases_ran] = self.tlm[self._ch_test_cases_ran] + 1

    def start(self) -> None:
        self._writer.start()

    def stop(self, timeout: float = 5.0) -> bool:
        return self._writer.stop(timeout)
