#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import dataclasses
import logging
import threading
from collections.abc import Callable
from typing import Literal

from pydantic import BaseModel

from freighter import UnaryClient
from freighter.transport import Empty
from synnax.util.send_required import send_required
from x.telem import CrudeTimeSpan, TimeSpan, TimeStamp
from x.telem.clock_skew import ClockSkewCalculator

logger = logging.getLogger(__name__)

Status = Literal["disconnected", "connecting", "connected", "failed"]


@dataclasses.dataclass
class State:
    status: Status = "disconnected"
    error: Exception | None = None
    message: str = "Disconnected"
    cluster_key: str = ""
    client_version: str = ""
    client_server_compatible: bool = False
    node_version: str = ""
    clock_skew: TimeSpan = dataclasses.field(default_factory=lambda: TimeSpan(0))
    clock_skew_exceeded: bool = False


class CheckResponse(BaseModel):
    cluster_key: str = ""
    node_version: str = ""
    node_time: TimeStamp = TimeStamp(0)


def _parse_version(v: str) -> tuple[int, int]:
    try:
        parts = v.split(".")
        return int(parts[0]), int(parts[1])
    except (IndexError, ValueError):
        return 0, 0


def _versions_compatible(v1: str, v2: str) -> bool:
    return _parse_version(v1) == _parse_version(v2)


def _client_is_newer(client_version: str, node_version: str) -> bool:
    return _parse_version(client_version) > _parse_version(node_version)


_TROUBLESHOOTING_URL = (
    "https://docs.synnaxlabs.com/reference/client/resources/troubleshooting"
)


def _create_version_warning(
    node_version: str | None,
    client_version: str,
    client_is_newer: bool,
) -> str:
    to_upgrade = "Core" if client_is_newer else "client"
    nv = f"{node_version} " if node_version is not None else ""
    age = "old" if client_is_newer else "new"
    return (
        f"The Synnax core version {nv}is too {age} for client version "
        f"{client_version}. This may cause compatibility issues. We recommend "
        f"updating the {to_upgrade}. For more information, see "
        f"{_TROUBLESHOOTING_URL}#old-{to_upgrade}-version"
    )


class Checker:
    DEFAULT = State()

    def __init__(
        self,
        client: UnaryClient,
        poll_freq: CrudeTimeSpan = TimeSpan.SECOND * 30,
        client_version: str = "",
        name: str | None = None,
        clock_skew_threshold: CrudeTimeSpan = TimeSpan.SECOND,
    ) -> None:
        self._client = client
        self._poll_freq = TimeSpan(poll_freq)
        self._client_version = client_version
        self._name = name
        self._clock_skew_threshold = TimeSpan(clock_skew_threshold)
        self._skew_calc = ClockSkewCalculator()
        self._state = State(client_version=client_version)
        self._lock = threading.Lock()
        self._on_change_handlers: list[Callable[[State], None]] = []
        self._stop_event = threading.Event()
        self._version_warned = False
        self.check()
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()

    def stop(self) -> None:
        self._stop_event.set()

    def check(self) -> State:
        with self._lock:
            prev_status = self._state.status
            prev_skew_exceeded = self._state.clock_skew_exceeded

        self._skew_calc.start()
        try:
            res = send_required(
                self._client, "/connectivity/check", Empty(), CheckResponse
            )
        except Exception as e:
            with self._lock:
                self._state.status = "failed"
                self._state.error = e
                self._state.message = str(e)
                state = dataclasses.replace(self._state)
        else:
            self._skew_calc.end(res.node_time)
            with self._lock:
                self._state.clock_skew = self._skew_calc.skew
                self._state.clock_skew_exceeded = self._skew_calc.exceeds(
                    self._clock_skew_threshold
                )
                if self._state.clock_skew_exceeded:
                    direction = (
                        "ahead of" if int(self._skew_calc.skew) > 0 else "behind"
                    )
                    logger.warning(
                        "Measured excessive clock skew between this host and "
                        "the Synnax core. This host is %s the Synnax core "
                        "by approximately %s.",
                        direction,
                        abs(self._skew_calc.skew),
                    )
                node_version = res.node_version
                if not node_version:
                    self._state.client_server_compatible = False
                    if not self._version_warned:
                        logger.warning(
                            _create_version_warning(
                                None, self._client_version, True
                            )
                        )
                        self._version_warned = True
                elif not _versions_compatible(self._client_version, node_version):
                    self._state.client_server_compatible = False
                    if not self._version_warned:
                        logger.warning(
                            _create_version_warning(
                                node_version,
                                self._client_version,
                                _client_is_newer(
                                    self._client_version, node_version
                                ),
                            )
                        )
                        self._version_warned = True
                else:
                    self._state.client_server_compatible = True
                self._state.status = "connected"
                self._state.message = f"Connected to {self._name or 'cluster'}"
                self._state.cluster_key = res.cluster_key
                self._state.node_version = res.node_version
                state = dataclasses.replace(self._state)

        changed = (
            prev_status != state.status
            or prev_skew_exceeded != state.clock_skew_exceeded
        )
        if changed and self._on_change_handlers:
            for handler in self._on_change_handlers:
                handler(state)
        return state

    @property
    def state(self) -> State:
        with self._lock:
            return dataclasses.replace(self._state)

    def on_change(self, callback: Callable[[State], None]) -> None:
        self._on_change_handlers.append(callback)

    def _run(self) -> None:
        poll_seconds = self._poll_freq.seconds
        while not self._stop_event.wait(timeout=poll_seconds):
            self.check()
