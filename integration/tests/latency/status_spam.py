#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Memory spike stress test — generic StatusService.Set spam (no driver needed).

Hypothesis:
    The memory leak is caused by ANY high-frequency StatusService.Set call, not just
    driver warnings. Each StatusService.Set call triggers:
        1. RBAC → access.Enforce() → traverseByScan (O(N) full KV scan)
        2. Status KV write → storeState.Copy (17.5% of customer allocs)
        3. Ontology KV write → storeState.Copy
        4. Bleve search index update (1.5%)

    In the customer's case it happened to be driver send_warning() at 68 Hz per task,
    but this test proves the leak is reproducible from pure Python StatusService.Set
    calls — no driver, no simulators, no hardware.

Setup sequence:
    1.  [Claude] rm -rf synnax-data/
    2.  [Claude] GOGC=3000 GOMAXPROCS=2 go run -tags driver core/main.go start -i --debug
    3.  [Script] Build ontology: bulk channels + calc + ranges + fake NI tasks
    4.  [Script] Spam client.statuses.set() targeting random ontology objects
    5.  [Script] Monitor RSS every 1s for DURATION seconds

Run from integration/:
    uv run python -m tests.latency.status_spam

Env vars:
    DURATION=300        Total test duration in seconds
    SPAM_RATE=70        Statuses per second (~68/s matches customer driver rate)
    N_THREADS=2         Spammer threads (needed to overcome ~22ms round-trip)
    N_STATUSES=5        Number of fixed statuses to cycle through
    N_INDEX=4           Index channels (x N_DATA = bulk channel count)
    N_DATA=50           Data channels per index (default 50 = 200 total)
    N_CALC=50           Calc channel count
    N_RANGES=50         Range count
    N_FAKE_NI=5         Fake NI task records (ontology weight)
    SKIP_BUILD=0        Set 1 to reuse existing entities
    SYNNAX_HOST         Server host (default localhost)
    SYNNAX_PORT         Server port (default 9090)
"""

import os
import random
import subprocess
import threading
import time
from datetime import datetime, timedelta, timezone

import synnax as sy
from synnax.ontology import ID
from synnax.status.payload import Status

from framework.test_case import SynnaxConnection, TestCase

PREFIX = "sts_spam_"

# Target ~68 calls/s to match customer's driver warning rate (68 Hz sample rate
# on degraded OPC UA tasks). Single-threaded Python tops out at ~46/s due to
# ~22ms round-trip per call, so we use 2 threads to reach the target.
DURATION = int(os.environ.get("DURATION", 300))
SPAM_RATE = int(os.environ.get("SPAM_RATE", 70))
N_THREADS = int(os.environ.get("N_THREADS", 2))
N_INDEX = int(os.environ.get("N_INDEX", 4))
N_DATA = int(os.environ.get("N_DATA", 50))
N_CALC = int(os.environ.get("N_CALC", 50))
N_RANGES = int(os.environ.get("N_RANGES", 50))
N_FAKE_NI = int(os.environ.get("N_FAKE_NI", 5))
SKIP_BUILD = os.environ.get("SKIP_BUILD", "0") == "1"
HOST = os.environ.get("SYNNAX_HOST", "localhost")
PORT = int(os.environ.get("SYNNAX_PORT", 9090))

N_STATUSES = int(os.environ.get("N_STATUSES", 5))


def _find_server_pid() -> int | None:
    try:
        r = subprocess.run(
            ["lsof", "-i", f":{PORT}", "-sTCP:LISTEN"],
            capture_output=True, text=True, timeout=5,
        )
        lines = r.stdout.strip().split("\n")
        if len(lines) > 1:
            return int(lines[1].split()[1])
    except Exception:
        pass
    return None


def _rss_mb(pid: int | None) -> int | None:
    if pid is None:
        return None
    try:
        r = subprocess.run(
            ["ps", "-o", "rss=", "-p", str(pid)],
            capture_output=True, text=True, timeout=2,
        )
        return int(r.stdout.strip()) // 1024
    except Exception:
        return None


def _total_ram_mb() -> int:
    try:
        r = subprocess.run(
            ["sysctl", "-n", "hw.memsize"],
            capture_output=True, text=True, timeout=2,
        )
        return int(r.stdout.strip()) // (1024 * 1024)
    except Exception:
        return 0


class StatusSpam(TestCase):
    """Memory spike stress test via pure StatusService.Set spam.

    No driver, no simulators — Python client calling
    client.statuses.set() at ~68/s (matching customer driver rate)
    targeting random ontology objects (channels, ranges, tasks).
    Each call is a separate RBAC check on the server. Uses N_THREADS
    threads to overcome Python's ~22ms per-call round-trip limit.

    Proves the leak is in the StatusService.Set → RBAC → traverseByScan
    path, independent of the driver warning mechanism.
    """

    def setup(self) -> None:
        super().setup()
        self._channel_keys: list[int] = []
        self._range_keys: list[str] = []
        self._task_keys: list[int] = []
        self._all_targets: list[ID] = []
        self._statuses: list[tuple[Status, ID]] = []
        self._build_entities()
        self._build_target_pool()
        self._build_status_pool()

    def _build_entities(self) -> None:
        if SKIP_BUILD:
            self.log("SKIP_BUILD=1 -- reusing existing entities")
            self._load_existing_keys()
            return

        t0 = time.time()
        self.log(
            f"Building ontology: {N_INDEX * N_DATA} data + {N_CALC} calc "
            f"channels, {N_RANGES} ranges, {N_FAKE_NI} tasks"
        )

        # Clean up old tasks with our prefix
        existing_tasks = [
            t for t in self.client.tasks.list() if t.name.startswith(PREFIX)
        ]
        if existing_tasks:
            self.client.tasks.delete([t.key for t in existing_tasks])

        # Index channels
        index_chs = self.client.channels.create(
            [
                sy.Channel(
                    name=f"{PREFIX}time_{i}",
                    data_type=sy.DataType.TIMESTAMP,
                    is_index=True,
                )
                for i in range(N_INDEX)
            ],
            retrieve_if_name_exists=True,
        )

        # Data channels
        data_chs = self.client.channels.create(
            [
                sy.Channel(
                    name=f"{PREFIX}data_{i}_{j}",
                    data_type=sy.DataType.FLOAT32,
                    index=index_chs[i].key,
                )
                for i in range(N_INDEX)
                for j in range(N_DATA)
            ],
            retrieve_if_name_exists=True,
        )
        self._channel_keys = [ch.key for ch in index_chs] + [
            ch.key for ch in data_chs
        ]

        # Calc channels
        self.client.channels.create(
            [
                sy.Channel(
                    name=f"{PREFIX}calc_{k}",
                    data_type=sy.DataType.FLOAT32,
                    expression=f"return {PREFIX}data_0_{k % N_DATA} + 1",
                )
                for k in range(N_CALC)
            ],
            retrieve_if_name_exists=True,
        )

        # Ranges
        now = datetime.now(tz=timezone.utc)
        for r in range(N_RANGES):
            start_dt = now - timedelta(days=N_RANGES - r)
            end_dt = start_dt + timedelta(hours=8)
            rng = self.client.ranges.create(
                name=f"{PREFIX}range_{r}",
                time_range=sy.TimeRange(
                    int(start_dt.timestamp() * 1e9),
                    int(end_dt.timestamp() * 1e9),
                ),
                retrieve_if_name_exists=True,
            )
            self._range_keys.append(str(rng.key))

        # Fake NI tasks (ontology weight)
        for i in range(N_FAKE_NI):
            tsk = self.client.tasks.create(
                sy.Task(
                    name=f"{PREFIX}zz_ni_task_{i}",
                    type="ni_analog_read",
                    config={},
                )
            )
            self._task_keys.append(tsk.key)

        elapsed = time.time() - t0
        self.log(f"  Ontology built in {elapsed:.1f}s")

    def _load_existing_keys(self) -> None:
        """Load keys from existing entities when SKIP_BUILD=1."""
        all_chs = self.client.channels.retrieve(
            search_term=PREFIX, limit=10000
        )
        self._channel_keys = [ch.key for ch in all_chs]

        all_ranges = self.client.ranges.search(PREFIX, limit=10000)
        self._range_keys = [str(r.key) for r in all_ranges]

        all_tasks = [
            t for t in self.client.tasks.list() if t.name.startswith(PREFIX)
        ]
        self._task_keys = [t.key for t in all_tasks]

        self.log(
            f"  Loaded {len(self._channel_keys)} channels, "
            f"{len(self._range_keys)} ranges, "
            f"{len(self._task_keys)} tasks"
        )

    def _build_target_pool(self) -> None:
        """Build the pool of ontology IDs to target with statuses."""
        for key in self._channel_keys:
            self._all_targets.append(ID(type="channel", key=str(key)))
        for key in self._range_keys:
            self._all_targets.append(ID(type="range", key=key))
        for key in self._task_keys:
            self._all_targets.append(ID(type="task", key=str(key)))

        if not self._all_targets:
            raise RuntimeError(
                "No ontology targets found — build entities first "
                "(set SKIP_BUILD=0)"
            )

        self.log(
            f"  Target pool: {len(self._all_targets)} ontology objects "
            f"({len(self._channel_keys)} ch + "
            f"{len(self._range_keys)} rng + "
            f"{len(self._task_keys)} tsk)"
        )

    def _build_status_pool(self) -> None:
        """Pre-create a small fixed set of statuses to cycle through.

        Realistic: a driver in a degraded state spams the same few
        warnings repeatedly, not unique messages each time.
        """
        templates = [
            ("warning", "failed to read data from device"),
            ("warning", "array size mismatch"),
            ("error", "device disconnected"),
            ("warning", "heartbeat missed"),
            ("info", "task encountered recoverable error"),
        ]

        for i in range(N_STATUSES):
            target = random.choice(self._all_targets)
            variant, msg = templates[i % len(templates)]
            self._statuses.append((
                Status(
                    variant=variant,
                    message=f"[{target.type}:{target.key}] {msg}",
                    name=f"{PREFIX}status_{i}",
                ),
                target,
            ))

        self.log(
            f"  Status pool: {len(self._statuses)} fixed statuses "
            f"(cycling through repeatedly)"
        )

    def _spammer_loop(
        self, thread_id: int, interval: float, stop_event: threading.Event
    ) -> None:
        """Spam loop for a single thread. Each thread gets its own client."""
        thread_client = sy.Synnax(
            host=HOST, port=PORT, username="synnax", password="seldon"
        )
        local_sent = 0

        try:
            while not stop_event.is_set():
                status, target = self._statuses[
                    local_sent % len(self._statuses)
                ]

                call_start = time.time()
                try:
                    thread_client.statuses.set(status, parent=target)
                    local_sent += 1
                    with self._counter_lock:
                        self._sent += 1
                except Exception:
                    with self._counter_lock:
                        self._errors += 1

                sleep_remaining = interval - (time.time() - call_start)
                if sleep_remaining > 0:
                    time.sleep(sleep_remaining)
        finally:
            thread_client.close()
            self.log(
                f"  Thread {thread_id}: {local_sent} sent"
            )

    def run(self) -> None:
        server_pid = _find_server_pid()
        baseline_rss = _rss_mb(server_pid)

        # Subscribe to mem% channel
        MEM_CH = "sy_node_1_metrics_mem_percentage"
        has_mem_ch = False
        try:
            self.subscribe(MEM_CH, timeout=5)
            has_mem_ch = True
        except Exception:
            self.log(f"  Warning: {MEM_CH} not found, skipping mem%")

        total_ram = _total_ram_mb()
        baseline_mem: float | None = None
        if has_mem_ch:
            try:
                baseline_mem = self.get_value(MEM_CH)
            except Exception:
                pass

        rate_per_thread = SPAM_RATE / max(N_THREADS, 1)
        interval = 1.0 / rate_per_thread if rate_per_thread > 0 else 1.0

        self.log(f"Server PID={server_pid}  baseline RSS={baseline_rss} MB")
        self.log(
            f"Duration={DURATION}s  spam_rate={SPAM_RATE}/s  "
            f"threads={N_THREADS}  rate/thread={rate_per_thread:.1f}/s"
        )
        self.log(
            f"Ontology: {N_INDEX * N_DATA} data + {N_CALC} calc channels, "
            f"{N_RANGES} ranges, {N_FAKE_NI} tasks"
        )

        # Shared counters for spammer threads
        self._sent = 0
        self._errors = 0
        self._counter_lock = threading.Lock()
        stop_event = threading.Event()

        # Launch spammer threads
        threads: list[threading.Thread] = []
        for i in range(N_THREADS):
            t = threading.Thread(
                target=self._spammer_loop,
                args=(i, interval, stop_event),
                daemon=True,
            )
            t.start()
            threads.append(t)

        self.log(f"Started {N_THREADS} spammer threads")

        # ── RSS monitoring loop ───────────────────────────────────────────

        start = time.time()
        last_report = start

        self.log("")
        self.log("  [elapsed]  RSS (delta)              mem%")
        self.log("  " + "-" * 50)

        mem_baseline_str = ""
        if baseline_mem is not None:
            mem_baseline_str = f"  mem={baseline_mem:.1f}% (+0 MB)"
        self.log(
            f"  [    0s]  RSS={baseline_rss} (+0 MB){mem_baseline_str}"
        )

        try:
            while time.time() - start < DURATION:
                time.sleep(1)
                now_time = time.time()
                elapsed = now_time - start

                rss = _rss_mb(server_pid)
                rss_delta = (
                    f" ({rss - baseline_rss:+d} MB)"
                    if (rss and baseline_rss)
                    else ""
                )
                mem_str = ""
                if has_mem_ch:
                    try:
                        mem_pct = self.get_value(MEM_CH)
                        if mem_pct is not None:
                            mem_delta_mb = 0
                            if baseline_mem is not None and total_ram > 0:
                                mem_delta_mb = int(
                                    (mem_pct - baseline_mem)
                                    / 100.0
                                    * total_ram
                                )
                            mem_str = (
                                f"  mem={mem_pct:.1f}% "
                                f"({mem_delta_mb:+d} MB)"
                            )
                    except Exception:
                        pass
                self.log(
                    f"  [{elapsed:5.0f}s]  "
                    f"RSS={rss}{rss_delta}{mem_str}  "
                    f"sent={self._sent}"
                )

        except KeyboardInterrupt:
            self.log("Interrupted by user")
        finally:
            stop_event.set()
            for t in threads:
                t.join(timeout=5)

        # ── Final summary ─────────────────────────────────────────────────

        end_rss = _rss_mb(server_pid)
        growth = (
            (end_rss - baseline_rss) if (end_rss and baseline_rss) else None
        )

        self.log("")
        self.log("=" * 65)
        self.log("StatusService.Set spam stress complete (no driver)")
        self.log("=" * 65)
        self.log(f"  RSS start  : {baseline_rss} MB")
        self.log(f"  RSS end    : {end_rss} MB")
        if growth is not None:
            trend = "GROWING" if growth > 500 else "STABLE"
            self.log(f"  RSS growth : {growth:+d} MB  {trend}")
        self.log(f"  Sent       : {self._sent} statuses ({self._errors} errors)")
        self.log(f"  Spam rate  : {SPAM_RATE}/s (target, {N_THREADS} threads)")
        actual_rate = self._sent / DURATION if DURATION > 0 else 0
        self.log(f"  Actual rate: {actual_rate:.1f}/s")
        self.log(
            f"  Targets    : {len(self._all_targets)} ontology objects"
        )
        self.log(f"  Duration   : {DURATION}s")

    def teardown(self) -> None:
        if not SKIP_BUILD:
            try:
                existing = [
                    t for t in self.client.tasks.list()
                    if t.name.startswith(PREFIX)
                ]
                if existing:
                    self.client.tasks.delete([t.key for t in existing])
            except Exception:
                pass
        super().teardown()


if __name__ == "__main__":
    test = StatusSpam(
        SynnaxConnection(server_address=HOST, port=PORT),
        name="status_spam",
    )
    test.execute()
