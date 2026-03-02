"""
RBAC-path storm — exercises the Console-like API call pattern.

Every call to channels.retrieve(), tasks.list(), and tasks.retrieve()
goes through RBAC enforcement in policy/retriever.go, which calls
TraverseTo(ParentsTraverser) — a full O(N) KV scan — for every
authenticated request, where N = number of ontology relationships.

This is DIFFERENT from 2_stress_traversal.py and 3_lifelike_storm.py,
which call the ontology API directly (retrieve_parents()). This script
drives the NORMAL API surface — the same HTTP endpoints the Console uses:

  POST /channel/retrieve  → RBAC → traverseByScan
  POST /task/retrieve     → RBAC → traverseByScan
  GET  /task/list         → RBAC → traverseByScan

Theory: Console activity alone (task panel refreshes, channel property
panels, status checks) sustains traverseByScan pressure. Under GC handicap
(GOGC=2000), the sustained background pressure produces measurable RSS growth
without any task restarts. When a restart happens on top, GC falls over.

Recommended server invocation (from repo root):
    GOGC=2000 GOMAXPROCS=3 go run -tags driver core/main.go start -i

Run from client/py/:
    uv run python examples/reproduce_mem_spike/4_scenario4_rbac_storm.py

Env vars:
    WORKERS=50         concurrent workers (default 50)
    DURATION=120       test duration in seconds (default 120)
    BURST=1            synchronized burst waves via Barrier (default 1; 0 = steady)
    BURST_GAP_MS=100   ms of rest between burst waves (default 100)
    SKIP_BUILD=0       set to 1 to skip entity build and reuse existing rbac_storm_ entities
    N_INDEX=8          number of index channels (default 8)
    N_DATA=25          data channels per index (default 25; total = N_INDEX × N_DATA)
    N_TASKS=5          number of fake task records (default 5)
    SYNNAX_HOST        server host (default localhost)
    SYNNAX_PORT        server port (default 9090)
"""

from __future__ import annotations

import os
import subprocess
import threading
import time
from concurrent.futures import ThreadPoolExecutor

import synnax as sy

# ─────────────────────────────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────────────────────────────

WORKERS = int(os.environ.get("WORKERS", 50))
DURATION = int(os.environ.get("DURATION", 120))
BURST_MODE = os.environ.get("BURST", "1") == "1"
BURST_GAP_MS = int(os.environ.get("BURST_GAP_MS", 100))
SKIP_BUILD = os.environ.get("SKIP_BUILD", "0") == "1"
HOST = os.environ.get("SYNNAX_HOST", "localhost")
PORT = int(os.environ.get("SYNNAX_PORT", 9090))

PREFIX = "rbac_storm_"

# Ontology size knobs — primary cost-per-call lever
# Default: customer-realistic (~200 channels, 5 tasks)
# Stress: N_INDEX=10 N_DATA=100 → 1,000 data channels
N_INDEX = int(os.environ.get("N_INDEX", 8))
N_DATA_PER_INDEX = int(os.environ.get("N_DATA", 25))  # N_INDEX × N_DATA = total data channels
N_FAKE_TASKS = int(os.environ.get("N_TASKS", 5))

# ─────────────────────────────────────────────────────────────────────────────
# RSS helpers (reused from other scripts)
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
# Phase 1 — Build entities (channels + task records)
# ─────────────────────────────────────────────────────────────────────────────


def build_entities(client: sy.Synnax) -> tuple[list[str], list[str]]:
    """Create channels and fake task records, return their names."""
    print("=" * 70)
    print("Phase 1 — Building RBAC-storm entities")
    print(f"  {N_INDEX} index channels × {N_DATA_PER_INDEX} data channels = "
          f"{N_INDEX * N_DATA_PER_INDEX} data channels")
    print(f"  {N_FAKE_TASKS} fake task records (no driver)")
    print("=" * 70)

    # Delete old rbac_storm_ tasks to avoid accumulation
    existing = [t for t in client.tasks.list() if t.name.startswith(PREFIX)]
    if existing:
        client.tasks.delete([t.key for t in existing])
        print(f"  Cleaned up {len(existing)} old {PREFIX} tasks")

    # Index channels
    index_channels: list[sy.Channel] = []
    for i in range(N_INDEX):
        ch = client.channels.create(
            name=f"{PREFIX}time_{i}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        index_channels.append(ch)

    # Data channels (batch create)
    client.channels.create(
        [
            sy.Channel(
                name=f"{PREFIX}data_{i}_{j}",
                data_type=sy.DataType.FLOAT32,
                index=index_channels[i].key,
            )
            for i in range(N_INDEX)
            for j in range(N_DATA_PER_INDEX)
        ],
        retrieve_if_name_exists=True,
    )

    # Fake task records — these exist as ontology nodes and are traversed
    # by RBAC just like real tasks, without requiring a running driver.
    fake_tasks: list[sy.task.Task] = []
    for i in range(N_FAKE_TASKS):
        tsk = client.tasks.create(
            name=f"{PREFIX}task_{i}",
            type="ni_analog_read",
            config={},
        )
        fake_tasks.append(tsk)

    channel_names = (
        [f"{PREFIX}time_{i}" for i in range(N_INDEX)]
        + [
            f"{PREFIX}data_{i}_{j}"
            for i in range(N_INDEX)
            for j in range(N_DATA_PER_INDEX)
        ]
    )
    task_names = [f"{PREFIX}task_{i}" for i in range(N_FAKE_TASKS)]

    n_channels = N_INDEX + N_INDEX * N_DATA_PER_INDEX
    print(f"\n  ✓ {n_channels} channels, {N_FAKE_TASKS} tasks ready")
    return channel_names, task_names


def load_entities(client: sy.Synnax) -> tuple[list[str], list[str]]:
    """Return entity names from existing rbac_storm_ resources."""
    print("SKIP_BUILD=1 — loading existing rbac_storm_ entities...")
    tasks = [t for t in client.tasks.list() if t.name.startswith(PREFIX)]
    channel_names = (
        [f"{PREFIX}time_{i}" for i in range(N_INDEX)]
        + [
            f"{PREFIX}data_{i}_{j}"
            for i in range(N_INDEX)
            for j in range(N_DATA_PER_INDEX)
        ]
    )
    task_names = [t.name for t in tasks]
    print(f"  ✓ {len(channel_names)} channel names, {len(task_names)} tasks")
    return channel_names, task_names


# ─────────────────────────────────────────────────────────────────────────────
# Phase 2 — Stress: RBAC-path retrieve calls
# ─────────────────────────────────────────────────────────────────────────────

_counter_lock = threading.Lock()
_total_calls = 0
_stop_event = threading.Event()


def _increment(n: int) -> None:
    global _total_calls
    with _counter_lock:
        _total_calls += n


def worker(
    worker_id: int,
    channel_names: list[str],
    task_names: list[str],
    barrier: threading.Barrier | None,
) -> int:
    """
    Calls normal API endpoints in a loop. Each call goes through RBAC enforcement
    which triggers TraverseTo(ParentsTraverser) — the same O(N) KV scan.

    Alternates between three call types to mimic Console behavior:
      - channels.retrieve() — channel service RBAC path
      - tasks.list()        — task service RBAC path (full list refresh)
      - tasks.retrieve()    — task service RBAC path (targeted retrieve)
    """
    wc = sy.Synnax(host=HOST, port=PORT, username="synnax", password="seldon")
    calls = 0
    call_type = worker_id % 3  # stagger call type across workers

    try:
        while not _stop_event.is_set():
            if barrier is not None:
                try:
                    barrier.wait(timeout=15.0)
                except threading.BrokenBarrierError:
                    break
                if _stop_event.is_set():
                    break

            try:
                if call_type == 0:
                    # Simulates Console channel property panel or line-plot channel list
                    wc.channels.retrieve(channel_names)
                elif call_type == 1:
                    # Simulates Console task management panel refresh
                    wc.tasks.list()
                else:
                    # Simulates Console task status poll
                    wc.tasks.retrieve(names=task_names)
            except Exception as e:
                if not _stop_event.is_set():
                    print(f"  [worker {worker_id}] error: {e}")

            calls += 1
            # Rotate call type on every iteration
            call_type = (call_type + 1) % 3

            if calls % 5 == 0:
                _increment(5)

            if BURST_MODE:
                time.sleep(BURST_GAP_MS / 1000.0)

    except Exception as e:
        print(f"  [worker {worker_id}] fatal: {e}")
    finally:
        _increment(calls % 5)
        wc.close()

    return calls


def stress_rbac(channel_names: list[str], task_names: list[str]) -> None:
    server_pid = _find_server_pid()
    baseline_rss = _rss_mb(server_pid)

    mode_str = (
        f"BURST (gap={BURST_GAP_MS} ms)" if BURST_MODE else "steady-state"
    )
    barrier = threading.Barrier(WORKERS) if BURST_MODE else None

    print("\n" + "=" * 70)
    print("Phase 2 — RBAC-path stress")
    print("=" * 70)
    print(f"  Mode       : {mode_str}")
    print(f"  Workers    : {WORKERS} (split across channels.retrieve / tasks.list / tasks.retrieve)")
    print(f"  Duration   : {DURATION}s")
    print(f"  Channels   : {len(channel_names)}")
    print(f"  Tasks      : {len(task_names)}")
    if baseline_rss:
        print(f"  Server     : PID={server_pid}  baseline RSS={baseline_rss} MB")
    print()
    print("NOTE: Each API call goes through RBAC → TraverseTo(ParentsTraverser)")
    print("      This is the path the Console drives, not the direct ontology API.")
    print()
    print("TIP: start server with GOGC=2000 GOMAXPROCS=3 for visible RSS growth")
    print()

    start_time = time.time()
    last_calls = 0

    executor = ThreadPoolExecutor(max_workers=WORKERS)
    futures = [
        executor.submit(worker, wid, channel_names, task_names, barrier)
        for wid in range(WORKERS)
    ]
    print(f"Starting {WORKERS} workers... stats every 5s.\n")

    try:
        while time.time() - start_time < DURATION:
            time.sleep(5)
            elapsed = time.time() - start_time
            with _counter_lock:
                current = _total_calls
            delta = current - last_calls
            calls_per_sec = delta / 5.0
            rss = _rss_mb(server_pid)
            rss_delta = (
                f" (+{rss - baseline_rss:+d} MB)" if (rss and baseline_rss) else ""
            )
            rss_str = f"  RSS={rss}{rss_delta}" if rss else ""
            print(
                f"  [{elapsed:5.0f}s] "
                f"{calls_per_sec:5.1f} API-calls/s"
                f"{rss_str} | "
                f"{current} total calls"
            )
            last_calls = current
    finally:
        _stop_event.set()
        if barrier is not None:
            barrier.abort()
        executor.shutdown(wait=True)

    total_elapsed = time.time() - start_time
    end_rss = _rss_mb(server_pid)
    rss_growth = (end_rss - baseline_rss) if (end_rss and baseline_rss) else None

    print("\n" + "=" * 70)
    print("Stress complete")
    print("=" * 70)
    print(f"  Mode          : {mode_str}")
    print(f"  Duration      : {total_elapsed:.1f}s")
    print(f"  Total calls   : {_total_calls}")
    print(f"  Avg calls/s   : {_total_calls / total_elapsed:.1f}")
    if baseline_rss:
        print(f"  RSS start     : {baseline_rss} MB")
    if end_rss:
        print(f"  RSS end       : {end_rss} MB")
    if rss_growth is not None:
        verdict = "✓ RSS GREW" if rss_growth > 100 else "— GC kept up"
        print(f"  RSS growth    : {rss_growth:+d} MB  {verdict}")

    print(
        "\nTo profile RBAC allocations (separate terminal while test runs):\n"
        f"  curl http://{HOST}:{PORT}/debug/pprof/allocs > allocs_rbac.bin\n"
        "  go tool pprof -text -sample_index=alloc_space allocs_rbac.bin | head -15\n"
        "\nExpect traverseByScan to appear as top allocator even though we\n"
        "called channels.retrieve() / tasks.list() — not retrieve_parents().\n"
        "That confirms the RBAC enforcement path is the actual root cause."
    )


# ─────────────────────────────────────────────────────────────────────────────
# Entry point
# ─────────────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    client = sy.Synnax(host=HOST, port=PORT, username="synnax", password="seldon")

    if SKIP_BUILD:
        ch_names, tsk_names = load_entities(client)
    else:
        ch_names, tsk_names = build_entities(client)

    stress_rbac(ch_names, tsk_names)
