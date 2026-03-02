"""
Stress-test the ontology traverseByScan path to reproduce the server memory spike.

Burst mode (default): a threading.Barrier synchronizes all workers so they fire
simultaneously in waves. This replicates the task-restart storm character — many
server goroutines hit traverseByScan at the same instant, flooding the allocator
faster than GC can collect. The live heap grows, pushing RSS up.

Steady-state mode (BURST=0): workers run independently. GC keeps up; RSS stays flat.
Use this mode for profiling the allocation cascade without RSS growth.

Run from client/py/ after 1_build_ontology.py:
    uv run python examples/reproduce_mem_spike/2_stress_traversal.py

Env vars:
    WORKERS=300      concurrent workers (default 300)
    DURATION=120     test duration in seconds (default 120)
    BURST=1          synchronized burst waves (default 1; set 0 for steady-state)
    BURST_GAP_MS=20  milliseconds of rest between burst waves (default 20)
    PPROF=0          capture pprof allocs snapshots (requires --debug on server)
    SYNNAX_HOST      server host (default localhost)
    SYNNAX_PORT      server port (default 9090)
"""

import os
import subprocess
import threading
import time
from concurrent.futures import ThreadPoolExecutor

import synnax as sy
from synnax.ontology.payload import ID

# ─────────────────────────────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────────────────────────────

WORKERS = int(os.environ.get("WORKERS", 300))
DURATION = int(os.environ.get("DURATION", 120))
BURST_MODE = os.environ.get("BURST", "1") == "1"
BURST_GAP_MS = int(os.environ.get("BURST_GAP_MS", 20))
HOST = os.environ.get("SYNNAX_HOST", "localhost")
PORT = int(os.environ.get("SYNNAX_PORT", 9090))
PPROF = os.environ.get("PPROF", "0") == "1"
PPROF_DIR = "pprof_snapshots"

# Must match values in 1_build_ontology.py
N_INDEX_CHANNELS = 50
N_DATA_CHANNELS_PER_INDEX = 100
CHANNEL_PREFIX = "repro_"

# ─────────────────────────────────────────────────────────────────────────────
# Server RSS helper
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
# Channel retrieval
# ─────────────────────────────────────────────────────────────────────────────

client = sy.Synnax(
    host=HOST, port=PORT, username="synnax", password="seldon"
)

print("Retrieving repro channels from server...")
expected_names = [
    f"{CHANNEL_PREFIX}data_{i}_{j}"
    for i in range(N_INDEX_CHANNELS)
    for j in range(N_DATA_CHANNELS_PER_INDEX)
]
try:
    all_channels = client.channels.retrieve(expected_names)
except Exception as e:
    print(
        f"ERROR: Could not retrieve repro channels: {e}\n"
        "Run 1_build_ontology.py first."
    )
    exit(1)
if not all_channels:
    print("ERROR: No repro channels found. Run 1_build_ontology.py first.")
    exit(1)

channel_ids = [ID(key=str(int(ch.key)), type="channel") for ch in all_channels]
server_pid = _find_server_pid()
baseline_rss = _rss_mb(server_pid)

print(f"  ✓ {len(channel_ids)} channels — ~{len(channel_ids) * 0.001:.0f} MB est alloc/call")
if baseline_rss:
    print(f"  ✓ Server PID={server_pid}  baseline RSS={baseline_rss} MB")

# ─────────────────────────────────────────────────────────────────────────────
# Shared state
# ─────────────────────────────────────────────────────────────────────────────

_counter_lock = threading.Lock()
_total_calls = 0
_stop_event = threading.Event()
_burst_barrier: threading.Barrier | None = (
    threading.Barrier(WORKERS) if BURST_MODE else None
)


def _increment(n: int) -> None:
    global _total_calls
    with _counter_lock:
        _total_calls += n


# ─────────────────────────────────────────────────────────────────────────────
# Worker
# ─────────────────────────────────────────────────────────────────────────────


def worker(worker_id: int, ids: list[ID]) -> int:
    """
    In burst mode: waits at the barrier so all WORKERS fire at the same instant,
    then rests for BURST_GAP_MS before the next wave. This maximises simultaneous
    goroutine allocation pressure on the server.
    """
    wc = sy.Synnax(
        host=HOST, port=PORT, username="synnax", password="seldon"
    )
    calls = 0
    idx = worker_id % len(ids)
    try:
        while not _stop_event.is_set():
            if _burst_barrier is not None:
                try:
                    _burst_barrier.wait(timeout=10.0)
                except threading.BrokenBarrierError:
                    break
                if _stop_event.is_set():
                    break
            wc.ontology.retrieve_parents(ids[idx % len(ids)])
            calls += 1
            idx += 1
            if calls % 5 == 0:
                _increment(5)
            if BURST_MODE:
                time.sleep(BURST_GAP_MS / 1000.0)
    except Exception as e:
        print(f"  [worker {worker_id}] error: {e}")
    finally:
        _increment(calls % 5)
        wc.close()
    return calls


# ─────────────────────────────────────────────────────────────────────────────
# pprof helper
# ─────────────────────────────────────────────────────────────────────────────


def capture_pprof(snapshot_n: int, profile: str = "allocs") -> None:
    if not PPROF:
        return
    import requests
    os.makedirs(PPROF_DIR, exist_ok=True)
    url = f"http://{HOST}:{PORT}/debug/pprof/{profile}"
    try:
        r = requests.get(url, timeout=10)
        r.raise_for_status()
        path = f"{PPROF_DIR}/{profile}_{snapshot_n:03d}.bin"
        with open(path, "wb") as f:
            f.write(r.content)
        print(f"    pprof → {path}  (parse: go tool pprof -text -sample_index=alloc_space {path} | head -10)")
    except Exception as e:
        print(f"    pprof failed: {e}")


# ─────────────────────────────────────────────────────────────────────────────
# Stats printer
# ─────────────────────────────────────────────────────────────────────────────


def print_stats(elapsed: float, last_calls: int, snapshot_n: int) -> tuple[int, int]:
    global _total_calls
    with _counter_lock:
        current = _total_calls
    delta = current - last_calls
    calls_per_sec = delta / 5.0
    est_mb_per_sec = calls_per_sec * len(channel_ids) * 0.001
    rss = _rss_mb(server_pid)
    rss_delta = f" (+{rss - baseline_rss:+d} MB)" if (rss and baseline_rss) else ""
    rss_str = f"  RSS={rss}{rss_delta}" if rss else ""
    print(
        f"  [{elapsed:5.0f}s] "
        f"{calls_per_sec:6.1f} calls/s | "
        f"~{est_mb_per_sec:6.1f} MB/s alloc (est)"
        f"{rss_str} | "
        f"{current} total calls"
    )
    if PPROF and int(elapsed) % 10 < 5:
        capture_pprof(snapshot_n, "allocs")
        snapshot_n += 1
    return current, snapshot_n


# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

mode_str = f"BURST (gap={BURST_GAP_MS} ms between waves)" if BURST_MODE else "steady-state"
print(f"\n{'=' * 70}")
print(f"Stress test: {WORKERS} workers × {DURATION}s — {mode_str}")
print(f"Channels: {len(channel_ids)} | ~{len(channel_ids) * 0.001:.0f} MB alloc/call | "
      f"{WORKERS} simultaneous goroutines on server per wave")
print(f"{'=' * 70}\n")

if PPROF:
    capture_pprof(0, "allocs")

print(f"Starting {WORKERS} workers (establishing connections)...")
start_time = time.time()
last_calls = 0
snapshot_n = 1

executor = ThreadPoolExecutor(max_workers=WORKERS)
futures = [executor.submit(worker, wid, channel_ids) for wid in range(WORKERS)]

print("Workers running. Stats every 5 seconds.\n")

try:
    while time.time() - start_time < DURATION:
        time.sleep(5)
        elapsed = time.time() - start_time
        last_calls, snapshot_n = print_stats(elapsed, last_calls, snapshot_n)
finally:
    _stop_event.set()
    if _burst_barrier is not None:
        _burst_barrier.abort()
    executor.shutdown(wait=True)

# ─────────────────────────────────────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────────────────────────────────────

total_elapsed = time.time() - start_time
end_rss = _rss_mb(server_pid)
rss_growth = (end_rss - baseline_rss) if (end_rss and baseline_rss) else None

print(f"\n{'=' * 70}")
print("Stress test complete")
print(f"{'=' * 70}")
print(f"  Mode             : {mode_str}")
print(f"  Duration         : {total_elapsed:.1f}s")
print(f"  Total calls      : {_total_calls}")
print(f"  Avg calls/s      : {_total_calls / total_elapsed:.1f}")
print(f"  Ontology size    : {len(channel_ids)} channels queried")
est_total_mb = _total_calls * len(channel_ids) * 0.001
print(f"  Est. total alloc : ~{est_total_mb:.0f} MB (before GC)")
if baseline_rss:
    print(f"  Server RSS start : {baseline_rss} MB")
if end_rss:
    print(f"  Server RSS end   : {end_rss} MB")
if rss_growth is not None:
    print(f"  RSS growth       : {rss_growth:+d} MB")

if PPROF:
    capture_pprof(snapshot_n, "allocs")
    print(f"\nCompare pprof snapshots in {PPROF_DIR}/ to confirm traverseByScan is top allocator.")
else:
    print(
        "\nTo capture a profile mid-test (separate terminal):\n"
        f"  curl http://{HOST}:{PORT}/debug/pprof/allocs > allocs_stress.bin\n"
        "  go tool pprof -text -sample_index=alloc_space allocs_stress.bin | head -15"
    )
