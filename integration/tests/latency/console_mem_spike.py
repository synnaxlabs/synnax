#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Memory spike stress test — real data sources + Playwright Console.

Scenario:
    - OPC UA + Modbus sims with 4 read + 2 write tasks (started by Claude)
    - 1000 bulk channels + 50 calc channels + 50 ranges with sub-ranges (built by script)
    - 5 fake NI tasks visible in the task toolbar (built by script)
    - Live line plot on the OPC UA / Modbus channels with a Rolling-30s range
    - Range Explorer snapped right (RBAC-gated range list queries)
    - Schematic with setpoint (write channel) + value (read channel) snapped far right
    - Full page reloads every RELOAD_INTERVAL_S seconds (default 1s — keeps GC busy)
    - 5 background streamers on live task channels (open_streamer — lifelike)

Setup sequence (Claude runs steps 1-12 via Bash; this script handles 13-18):
    1.  [Claude] rm -rf synnax-data/  (wipe server data for a fresh start)
    2.  [Claude] GOGC=3000 GOMAXPROCS=3 go run -tags driver core/main.go start -i
    3.  [Claude] cd client/py && uv run python -m examples.opcua.server &
    4.  [Claude] cd client/py && uv run python -m examples.modbus.server &
    5.  [Claude] echo "y" | uv run python -m examples.opcua.connect_server
    6.  [Claude] echo "y" | uv run python -m examples.modbus.connect_server
    7.  [Claude] cd client/py && uv run python -m examples.opcua.read_task &
    8.  [Claude] cd client/py && uv run python -m examples.opcua.read_task_array &
    9.  [Claude] cd client/py && uv run python -m examples.opcua.read_task_boolean &
    10. [Claude] cd client/py && uv run python -m examples.modbus.read_task &
    11. [Claude] cd client/py && uv run python -m examples.opcua.write_task &
    12. [Claude] cd client/py && uv run python -m examples.modbus.write_task &
    13. [Script] Build 1000 bulk channels + 50 calc + 50 ranges + 5 tasks  (SKIP_BUILD=0)
    14. [Script] Open Playwright browser (≤15s after entity build)
    15. [Script] Range Explorer (snapped right)
    16. [Script] Schematic with setpoint + value symbols (snapped far right)
    17. [Script] 5 background streamers on live task channels
    18. [Script] Reload Console every RELOAD_INTERVAL_S seconds (default 1s), log RSS

Also required before running:
    - Console Vite dev server:
         cd console && pnpm dev:console-vite
    - Run from integration/:
         PLAYWRIGHT_CONSOLE_HEADED=1 uv run python tests/latency/console_mem_spike.py

Env vars:
    PLAYWRIGHT_CONSOLE_HEADED=1   Run with browser visible (set this)
    DURATION=600                  Total test duration in seconds (default 600)
    RELOAD_INTERVAL_S=1           Seconds between full page reloads (default 1)
    N_INDEX=20                    Index channel count (default 20)
    N_DATA=50                     Data channels per index (default 50, = 1000 total)
    N_CALC=50                     Calc channel count (default 50)
    N_RANGES=50                   Range count, each with 2 sub-ranges (default 50)
    N_TASKS=5                     Number of fake NI task records (default 5)
    API_WORKERS=5                 Background streamer count (default 5)
    SKIP_BUILD=0                  Set 1 to skip entity build and reuse existing ones
    SYNNAX_HOST                   Server host (default localhost)
    SYNNAX_PORT                   Server port (default 9090)
"""

import os
import subprocess
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timedelta, timezone

import synnax as sy

from console.case import ConsoleCase
from console.schematic.setpoint import Setpoint
from console.schematic.value import Value
from framework.test_case import SynnaxConnection

PREFIX = "mem_spike_"

DURATION = int(os.environ.get("DURATION", 600))
RELOAD_INTERVAL_S = int(os.environ.get("RELOAD_INTERVAL_S", 1))
N_INDEX = int(os.environ.get("N_INDEX", 20))
N_DATA = int(os.environ.get("N_DATA", 50))
N_CALC = int(os.environ.get("N_CALC", 50))
N_RANGES = int(os.environ.get("N_RANGES", 50))
N_TASKS = int(os.environ.get("N_TASKS", 5))
API_WORKERS = int(os.environ.get("API_WORKERS", 5))
SKIP_BUILD = os.environ.get("SKIP_BUILD", "0") == "1"
HOST = os.environ.get("SYNNAX_HOST", "localhost")
PORT = int(os.environ.get("SYNNAX_PORT", 9090))


# ─────────────────────────────────────────────────────────────────────────────
# RSS helpers
# ─────────────────────────────────────────────────────────────────────────────


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


# ─────────────────────────────────────────────────────────────────────────────
# Test case
# ─────────────────────────────────────────────────────────────────────────────


class ConsoleMemSpike(ConsoleCase):
    """Memory spike stress test with real OPC UA + Modbus data sources.

    Setup (before browser opens):
    - OPC UA + Modbus sims + 4 read + 2 write tasks started externally by Claude
    - 1000 bulk channels + 50 calc channels + 50 ranges with sub-ranges
    - 5 fake NI tasks in the task toolbar

    Stress:
    - Playwright Console: task toolbar + line plot + range explorer + schematic + reloads
    - Schematic with setpoint (write channel) + value (read channel)
    - 5 background streamers on live OPC UA / Modbus channels
    """

    _data_channel_names: list[str]
    _task_names: list[str]
    _stop_workers: threading.Event
    _call_count: int
    _call_lock: threading.Lock

    def setup(self) -> None:
        self._cleanup_pages: list[str] = []
        self._build_entities()
        super().setup()
        self._stop_workers = threading.Event()
        self._call_count = 0
        self._call_lock = threading.Lock()

    def _build_entities(self) -> None:
        """Build channels/ranges/tasks. Sims must already be running externally."""
        if SKIP_BUILD:
            self.log("SKIP_BUILD=1 — reusing existing mem_spike_ entities")
            self._data_channel_names = [
                f"{PREFIX}data_{i}_{j}"
                for i in range(N_INDEX)
                for j in range(N_DATA)
            ]
            self._task_names = [f"{PREFIX}task_{i}" for i in range(N_TASKS)]
            return

        t0 = time.time()
        self.log(
            f"── Building ontology: {N_INDEX * N_DATA} data + {N_CALC} calc "
            f"channels, {N_RANGES} ranges, {N_TASKS} tasks ──"
        )

        # Clean up old mem_spike_ tasks
        existing_tasks = [
            t for t in self.client.tasks.list() if t.name.startswith(PREFIX)
        ]
        if existing_tasks:
            self.client.tasks.delete([t.key for t in existing_tasks])

        # Index channels (batch)
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

        # Data channels (single batch)
        self.client.channels.create(
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

        # Calc channels — expressions referencing real OPC UA channels (all f32).
        # Only float channels here; mixing u8 (Modbus) with f32 causes a type error.
        float_channels = ["my_float_0", "my_float_1"]
        self.client.channels.create(
            [
                sy.Channel(
                    name=f"{PREFIX}calc_{k}",
                    data_type=sy.DataType.FLOAT32,
                    expression=(
                        f"return {float_channels[k % len(float_channels)]} + "
                        f"{float_channels[(k + 1) % len(float_channels)]}"
                    ),
                )
                for k in range(N_CALC)
            ],
            retrieve_if_name_exists=True,
        )

        # Ranges with 2 sub-ranges each
        now = datetime.now(tz=timezone.utc)
        for r in range(N_RANGES):
            start_dt = now - timedelta(days=N_RANGES - r)
            end_dt = start_dt + timedelta(hours=8)
            parent = self.client.ranges.create(
                name=f"{PREFIX}range_{r}",
                time_range=sy.TimeRange(
                    int(start_dt.timestamp() * 1e9),
                    int(end_dt.timestamp() * 1e9),
                ),
                retrieve_if_name_exists=True,
            )
            parent.create_sub_range(
                name=f"{PREFIX}range_{r}_pre",
                time_range=sy.TimeRange(
                    int(start_dt.timestamp() * 1e9),
                    int((start_dt + timedelta(hours=2)).timestamp() * 1e9),
                ),
            )
            parent.create_sub_range(
                name=f"{PREFIX}range_{r}_run",
                time_range=sy.TimeRange(
                    int((start_dt + timedelta(hours=2)).timestamp() * 1e9),
                    int(end_dt.timestamp() * 1e9),
                ),
            )

        # Fake NI tasks (ontology entities — task toolbar)
        for i in range(N_TASKS):
            self.client.tasks.create(
                sy.Task(name=f"{PREFIX}task_{i}", type="ni_analog_read", config={})
            )

        self._data_channel_names = [
            f"{PREFIX}data_{i}_{j}"
            for i in range(N_INDEX)
            for j in range(N_DATA)
        ]
        self._task_names = [f"{PREFIX}task_{i}" for i in range(N_TASKS)]

        elapsed = time.time() - t0
        self.log(
            f"  ✓ {N_INDEX * N_DATA} data + {N_CALC} calc channels, "
            f"{N_RANGES} ranges (×3 with sub-ranges), {N_TASKS} tasks "
            f"— {elapsed:.1f}s total setup"
        )
        self.log(f"  → browser launching now (≤15s from entity build finish)")

    def _snap_tab_right(self, tab_name: str) -> None:
        """Snap an existing tab to the right side of the viewport."""
        pg = self.console.layout.page
        tab = self.console.layout.get_tab(tab_name)
        tab.wait_for(state="visible", timeout=5000)
        tab_box = tab.bounding_box()
        assert tab_box is not None
        viewport = pg.viewport_size
        assert viewport is not None
        start_x = tab_box["x"] + tab_box["width"] / 2
        start_y = tab_box["y"] + tab_box["height"] / 2
        drop_x = viewport["width"] - 100
        drop_y = viewport["height"] // 2
        pg.mouse.move(start_x, start_y)
        pg.mouse.down()
        pg.mouse.move(drop_x, drop_y, steps=10)
        time.sleep(0.2)
        pg.mouse.up()
        time.sleep(0.5)

    def _open_schematic_right(self) -> None:
        """Create a schematic with setpoint + value symbols, snapped right."""
        schematic = self.console.workspace.create_schematic(
            "MemSpike Schematic",
        )
        self._cleanup_pages.append("MemSpike Schematic")
        setpoint = schematic.create_symbol(
            Setpoint(label="OPC Cmd 0", channel_name="opcua_cmd_0"),
        )
        value = schematic.create_symbol(
            Value(label="Float 0", channel_name="my_float_0"),
        )
        value.move(delta_x=200, delta_y=0)
        schematic.disable_edit()
        schematic.move("right")

    def _reader_worker(self, worker_id: int) -> None:
        """Stream live task channels — mirrors what the Console does."""
        conn = self.synnax_connection
        wc = sy.Synnax(
            host=conn.server_address,
            port=conn.port,
            username=conn.username,
            password=conn.password,
            secure=conn.secure,
        )
        live_channels = ["my_float_0", "my_float_1", "input_register_0", "input_register_1"]
        try:
            with wc.open_streamer(live_channels) as streamer:
                while not self._stop_workers.is_set():
                    frame = streamer.read(timeout=1.0)
                    if frame is not None:
                        with self._call_lock:
                            self._call_count += 1
        except Exception:
            pass
        finally:
            wc.close()

    def run(self) -> None:
        server_pid = _find_server_pid()
        baseline_rss = _rss_mb(server_pid)
        self.log(f"Server PID={server_pid}  baseline RSS={baseline_rss} MB")
        self.log(
            f"Duration={DURATION}s  reload_every={RELOAD_INTERVAL_S}s  "
            f"readers={API_WORKERS}  data_channels={N_INDEX * N_DATA}  "
            f"calc={N_CALC}  ranges={N_RANGES}  tasks={N_TASKS}"
        )

        # ── UI Setup ─────────────────────────────────────────────────────────

        self.console.tasks.show_toolbar()
        for name in self._task_names:
            self.console.tasks.wait_for_task(name)
        self.log(f"Task toolbar showing {N_TASKS} tasks")

        # Use real live channels from OPC UA + Modbus read tasks
        plot_channels = ["my_float_0", "my_float_1", "input_register_0", "input_register_1"]
        plot = self.console.workspace.create_plot("MemSpike Live Plot")
        self._cleanup_pages.append("MemSpike Live Plot")
        plot.add_channels("Y1", plot_channels)
        self.log(f"Line plot open: {len(plot_channels)} channels, Rolling 30s")

        # Open Range Explorer snapped right — adds RBAC-gated range list queries
        self.console.ranges.open_explorer()
        self._snap_tab_right("Range Explorer")
        self.log("Range Explorer opened (snapped right)")

        # Open schematic with setpoint + value symbols, snapped to the far right
        self._open_schematic_right()
        self.log("Schematic with setpoint + value opened (snapped far right)")

        # ── Background readers ────────────────────────────────────────────────

        executor = ThreadPoolExecutor(max_workers=API_WORKERS)
        for wid in range(API_WORKERS):
            executor.submit(self._reader_worker, wid)
        self.log(f"Started {API_WORKERS} background reader workers")

        # ── Stress loop ───────────────────────────────────────────────────────

        start = time.time()
        last_report = start
        last_reload = start
        last_calls = 0
        reload_n = 0

        self.log("")
        self.log("  [elapsed]  RSS (delta)        | reloads | reads/s | notes")
        self.log("  " + "-" * 65)

        try:
            while time.time() - start < DURATION:
                now = time.time()
                elapsed = now - start

                if now - last_reload >= RELOAD_INTERVAL_S:
                    try:
                        self.console.reload()
                        self.console.notifications.close_connection()
                    except Exception:
                        pass
                    reload_n += 1
                    last_reload = time.time()
                    self.log(f"  [{elapsed:5.0f}s]  reloaded ({reload_n})")

                if now - last_report >= 5.0:
                    rss = _rss_mb(server_pid)
                    with self._call_lock:
                        calls_now = self._call_count
                    call_rate = (calls_now - last_calls) / (now - last_report)
                    last_calls = calls_now
                    rss_delta = (
                        f" (+{rss - baseline_rss:+d} MB)"
                        if (rss and baseline_rss)
                        else ""
                    )
                    self.log(
                        f"  [{elapsed:5.0f}s]  RSS={rss}{rss_delta:<20s} | "
                        f"reloads={reload_n:<4d} | "
                        f"reads={call_rate:4.1f}/s"
                    )
                    last_report = now

        finally:
            self._stop_workers.set()
            executor.shutdown(wait=False)

        # ── Final summary ─────────────────────────────────────────────────────

        end_rss = _rss_mb(server_pid)
        growth = (end_rss - baseline_rss) if (end_rss and baseline_rss) else None

        self.log("")
        self.log("=" * 65)
        self.log("Console mem spike stress complete")
        self.log("=" * 65)
        self.log(f"  RSS start  : {baseline_rss} MB")
        self.log(f"  RSS end    : {end_rss} MB")
        if growth is not None:
            trend = "↑ GROWING" if growth > 500 else "→ STABLE"
            self.log(f"  RSS growth : {growth:+d} MB  {trend}")
        self.log(f"  Reloads    : {reload_n}")
        self.log(f"  Reads      : {self._call_count}")
        self.log(f"  Duration   : {DURATION}s")

    def teardown(self) -> None:
        # Clean up mem_spike_ entities
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
    test = ConsoleMemSpike(
        SynnaxConnection(server_address=HOST, port=PORT),
        name="console_mem_spike",
        headed=True,
    )
    test.execute()
