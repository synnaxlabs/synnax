#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Regression test for SY-3812: lock-ordering deadlock in cesium/internal/control/.

The cesium control layer had a lock-ordering deadlock: OpenGate (writer open)
locked controller -> region, while release (writer close) locked region ->
controller. The fix moves controller.remove() outside the region lock in
release(), maintaining consistent lock ordering. A TOCTOU re-check in remove()
handles the race window where OpenGate could add a gate between the unlock
and removal.

This test hammers the server with concurrent transient writers (open/write/close)
on overlapping virtual channels. A background health probe detects any freeze
by attempting channels.create - the first operation to starve under the old bug.
"""

import threading

import synnax as sy

from framework.test_case import TestCase

PROBE_TIMEOUT = 2 * sy.TimeSpan.SECOND
NUM_THREADS = 20
OPS_PER_SEC = 100
TEST_DURATION = 10 * sy.TimeSpan.SECOND


class WriterOpenCloseStress(TestCase):
    """Regression test for SY-3812: lock-ordering deadlock in cesium control.

    Spawns N threads doing rapid transient open/write/close on overlapping virtual
    channels. Probes server health at 5 Hz via channels.create. Fails immediately
    if the server stops responding.
    """

    def run(self) -> None:
        self._channel_names = [
            self.client.channels.create(
                name=f"sy3812_deadlock_{i}",
                data_type=sy.DataType.FLOAT32,
                virtual=True,
                retrieve_if_name_exists=True,
            ).name
            for i in range(2)
        ]
        self.log(f"{len(self._channel_names)} virtual channels ready")

        self._interval = (
            sy.TimeSpan(int(1e9 / OPS_PER_SEC)) if OPS_PER_SEC > 0 else None
        )
        self._stop = threading.Event()
        threads: list[threading.Thread] = []

        # Start writer threads
        for _ in range(NUM_THREADS):
            t = threading.Thread(target=self._stress_loop, daemon=True)
            t.start()
            threads.append(t)

        rate = f"{OPS_PER_SEC} ops/s" if OPS_PER_SEC > 0 else "unlimited"
        self.log(
            f"{NUM_THREADS} writer threads active ({rate}), " f"probing at 5 Hz..."
        )

        timer = sy.Timer()
        while timer.elapsed() < TEST_DURATION:
            ok, latency = self._health_probe()
            elapsed_ms = int(timer.elapsed() / sy.TimeSpan.MILLISECOND)
            ts = f"{elapsed_ms / 1000:.2f}s"

            if not ok or latency > PROBE_TIMEOUT:
                self._stop.set()
                self.fail(
                    f"Server deadlock detected at {ts}. "
                    f"Health probe latency: {latency}."
                )
                return

            self.log(f"  [{ts}] OK latency={latency}")
            sy.sleep(200 * sy.TimeSpan.MILLISECOND)

        self._stop.set()
        for t in threads:
            t.join(timeout=2)

    def _stress_loop(self) -> None:
        """Rapid open/write/close on shared channels until told to stop."""
        conn = self.synnax_connection
        client = sy.Synnax(
            host=conn.server_address,
            port=conn.port,
            username=conn.username,
            password=conn.password,
            secure=conn.secure,
        )
        i = 0
        try:
            while not self._stop.is_set():
                ch = self._channel_names[i % len(self._channel_names)]
                op_timer = sy.Timer()
                w = client.open_writer(
                    start=sy.TimeStamp.now(),
                    channels=[ch],
                    enable_auto_commit=True,
                )
                w.write({ch: [float(i)]})
                w.close()
                i += 1
                if self._interval is not None:
                    remaining = self._interval - op_timer.elapsed()
                    if remaining > sy.TimeSpan(0):
                        sy.sleep(remaining)
        finally:
            client.close()

    def _health_probe(self) -> tuple[bool, sy.TimeSpan]:
        """Attempt channels.create - the operation blocked by the deadlock."""
        timer = sy.Timer()
        try:
            self.client.channels.create(
                name="__health_probe__",
                data_type=sy.DataType.FLOAT32,
                virtual=True,
                retrieve_if_name_exists=True,
            )
            return True, timer.elapsed()
        except Exception:
            return False, timer.elapsed()
