"""
Task churn storm — tests concurrent task create/delete cycles.

Each task create triggers a cascade of server-side allocations:
  1. RBAC traversal  : TraverseTo(ParentsTraverser) — O(N) KV scan
  2. DAG cycle check : DefineRelationship() in writer_dag.go
  3. Bleve index write: NewEmulatedBatch() in the search indexer

The critical variable is CONCURRENCY_WINDOW_MS:
  - 0 ms  (simultaneous): all N_TASKS creates fire at the same instant.
           All N cascades overlap in the heap. GC cannot collect one set
           of garbage before the next batch lands. Live heap spikes.
  - 5000 ms (sequential): creates are spread over 5 seconds.
           Each cascade runs and is collected before the next begins.
           GC keeps up. Minimal RSS growth.

This directly reproduces Scenario 1 from the runbook ("The Manual Restart"):
  User clicks "Start All Tasks" → 5 creates fire within a 2–3 second window
  → 15–20 concurrent traverseByScan + Bleve batches + DAG checks overwhelm GC.

The test runs N_CYCLES of: create N_TASKS simultaneously → pause → delete all.
Each cycle reports the RSS delta so accumulation across cycles is visible.

Recommended server invocation (from repo root):
    GOGC=2000 GOMAXPROCS=3 go run -tags driver core/main.go start -i

Run from client/py/:
    uv run python examples/reproduce_mem_spike/5_scenario1_task_churn_storm.py

Env vars:
    N_TASKS=5                    tasks per churn cycle (default 5, matches customer)
    N_CYCLES=20                  number of create+delete rounds (default 20)
    CONCURRENCY_WINDOW_MS=0      ms window over which creates are spread
                                 0 = simultaneous (worst case)
                                 5000 = sequential (GC-friendly baseline)
    PAUSE_BETWEEN_CYCLES_MS=500  rest between cycles (default 500)
    SYNNAX_HOST                  server host (default localhost)
    SYNNAX_PORT                  server port (default 9090)
"""

from __future__ import annotations

import os
import subprocess
import threading
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

import synnax as sy

# ─────────────────────────────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────────────────────────────

N_TASKS = int(os.environ.get("N_TASKS", 5))
N_CYCLES = int(os.environ.get("N_CYCLES", 20))
CONCURRENCY_WINDOW_MS = int(os.environ.get("CONCURRENCY_WINDOW_MS", 0))
PAUSE_BETWEEN_CYCLES_MS = int(os.environ.get("PAUSE_BETWEEN_CYCLES_MS", 500))
HOST = os.environ.get("SYNNAX_HOST", "localhost")
PORT = int(os.environ.get("SYNNAX_PORT", 9090))

PREFIX = "churn_storm_"

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
# Churn helpers
# ─────────────────────────────────────────────────────────────────────────────


def _create_one_task(
    client: sy.Synnax,
    task_idx: int,
    cycle: int,
    delay_s: float,
) -> sy.task.Task:
    """
    Creates a single fake task record after an optional delay.
    The delay implements CONCURRENCY_WINDOW_MS spreading.
    """
    if delay_s > 0:
        time.sleep(delay_s)
    return client.tasks.create(
        name=f"{PREFIX}cycle{cycle}_task{task_idx}",
        type="ni_analog_read",
        config={},
    )


def run_churn_cycle(
    client: sy.Synnax,
    cycle: int,
    server_pid: int | None,
    baseline_rss: int | None,
) -> None:
    """
    One churn cycle:
      1. Fire N_TASKS creates, spread over CONCURRENCY_WINDOW_MS.
         0 ms → all fire simultaneously (worst case).
         >0 ms → spread linearly (GC-friendly baseline for comparison).
      2. Wait for all creates to complete.
      3. Delete all created tasks.
      4. Pause PAUSE_BETWEEN_CYCLES_MS ms.
    """
    rss_before = _rss_mb(server_pid)

    # Compute per-task delay to spread creates over the concurrency window.
    # With 5 tasks and 0 ms window: all delays are 0 → simultaneous fire.
    # With 5 tasks and 5000 ms window: delays are 0, 1000, 2000, 3000, 4000 ms.
    if N_TASKS > 1 and CONCURRENCY_WINDOW_MS > 0:
        delay_step_s = (CONCURRENCY_WINDOW_MS / 1000.0) / (N_TASKS - 1)
    else:
        delay_step_s = 0.0

    # Fire all creates concurrently (the thread pool fires them at the same
    # real-time moment; per-task delays spread them within the window).
    created_tasks: list[sy.task.Task] = []
    create_start = time.time()

    with ThreadPoolExecutor(max_workers=N_TASKS) as ex:
        futures = {
            ex.submit(_create_one_task, client, i, cycle, i * delay_step_s): i
            for i in range(N_TASKS)
        }
        for fut in as_completed(futures):
            try:
                created_tasks.append(fut.result())
            except Exception as e:
                print(f"    [cycle {cycle}] create failed: {e}")

    create_elapsed_ms = (time.time() - create_start) * 1000
    rss_after_create = _rss_mb(server_pid)

    # Delete all created tasks
    if created_tasks:
        client.tasks.delete([t.key for t in created_tasks])

    rss_after_delete = _rss_mb(server_pid)

    # Report
    create_delta = (
        f" (+{rss_after_create - rss_before:+d} MB)"
        if (rss_after_create and rss_before)
        else ""
    )
    delete_delta = (
        f" (+{rss_after_delete - rss_before:+d} MB from baseline)"
        if (rss_after_delete and rss_before)
        else ""
    )
    cumulative = (
        f"  [cumulative: {rss_after_delete - baseline_rss:+d} MB from start]"
        if (rss_after_delete and baseline_rss)
        else ""
    )

    print(
        f"  Cycle {cycle:3d}/{N_CYCLES}"
        f"  created={len(created_tasks)}/{N_TASKS}"
        f"  window={create_elapsed_ms:.0f} ms"
        f"  RSS after-create={rss_after_create}{create_delta}"
        f"  after-delete={rss_after_delete}{delete_delta}"
        f"{cumulative}"
    )

    if PAUSE_BETWEEN_CYCLES_MS > 0:
        time.sleep(PAUSE_BETWEEN_CYCLES_MS / 1000.0)


# ─────────────────────────────────────────────────────────────────────────────
# Entry point
# ─────────────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    client = sy.Synnax(host=HOST, port=PORT, username="synnax", password="seldon")

    # Clean up any leftover churn_storm_ tasks from a previous run
    leftover = [t for t in client.tasks.list() if t.name.startswith(PREFIX)]
    if leftover:
        client.tasks.delete([t.key for t in leftover])
        print(f"Cleaned up {len(leftover)} leftover {PREFIX} tasks")

    server_pid = _find_server_pid()
    baseline_rss = _rss_mb(server_pid)

    concurrency_label = (
        "SIMULTANEOUS (0 ms window)"
        if CONCURRENCY_WINDOW_MS == 0
        else f"SPREAD over {CONCURRENCY_WINDOW_MS} ms"
    )

    print("\n" + "=" * 70)
    print("Task churn storm")
    print("=" * 70)
    print(f"  N_TASKS                : {N_TASKS} per cycle")
    print(f"  N_CYCLES               : {N_CYCLES}")
    print(f"  CONCURRENCY_WINDOW_MS  : {CONCURRENCY_WINDOW_MS} ms → {concurrency_label}")
    print(f"  PAUSE_BETWEEN_CYCLES   : {PAUSE_BETWEEN_CYCLES_MS} ms")
    if server_pid:
        print(f"  Server                 : PID={server_pid}  baseline RSS={baseline_rss} MB")
    print()
    print("Each create fires: RBAC traversal + DAG cycle check + Bleve index write")
    print(f"Each cycle fires {N_TASKS} creates {concurrency_label}")
    print()
    print("COMPARE MODES:")
    print(f"  Worst case  : CONCURRENCY_WINDOW_MS=0    → all {N_TASKS} overlap → GC falls behind")
    print(f"  GC-friendly : CONCURRENCY_WINDOW_MS=5000 → sequential → GC keeps up")
    print()
    print("TIP: start server with GOGC=2000 GOMAXPROCS=3 for visible RSS growth")
    print("=" * 70)
    print()

    start_time = time.time()

    for cycle in range(1, N_CYCLES + 1):
        run_churn_cycle(client, cycle, server_pid, baseline_rss)

    total_elapsed = time.time() - start_time
    end_rss = _rss_mb(server_pid)
    rss_growth = (end_rss - baseline_rss) if (end_rss and baseline_rss) else None

    print()
    print("=" * 70)
    print("Churn complete")
    print("=" * 70)
    print(f"  Total cycles           : {N_CYCLES}")
    print(f"  Total tasks created    : {N_CYCLES * N_TASKS}")
    print(f"  Concurrency mode       : {concurrency_label}")
    print(f"  Duration               : {total_elapsed:.1f}s")
    if baseline_rss:
        print(f"  RSS start              : {baseline_rss} MB")
    if end_rss:
        print(f"  RSS end                : {end_rss} MB")
    if rss_growth is not None:
        verdict = "✓ RSS GREW (GC behind)" if rss_growth > 50 else "— GC kept up"
        print(f"  RSS growth             : {rss_growth:+d} MB  {verdict}")

    print(
        "\nTo compare simultaneous vs sequential:"
        "\n  # Worst case (all tasks fire at once):"
        "\n  CONCURRENCY_WINDOW_MS=0 uv run python examples/reproduce_mem_spike/5_scenario1_task_churn_storm.py"
        "\n"
        "\n  # GC-friendly baseline (spread over 5 seconds):"
        "\n  CONCURRENCY_WINDOW_MS=5000 uv run python examples/reproduce_mem_spike/5_scenario1_task_churn_storm.py"
        "\n"
        "\nTo capture allocs profile during a run (separate terminal):"
        f"\n  curl http://{HOST}:{PORT}/debug/pprof/allocs > allocs_churn.bin"
        "\n  go tool pprof -text -sample_index=alloc_space allocs_churn.bin | head -15"
    )
