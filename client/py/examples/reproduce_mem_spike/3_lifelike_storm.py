"""
Lifelike task restart storm — models the customer environment.

Matches the approximate configuration from the customer report:
  - ~200 channels total:
      7 index channels × 24 data channels = 168 raw channels
      1 calculated index + 30 calculated channels
  - 5 tasks: 2 OPC UA (driver-validated), 3 NI analog read (raw creates, no driver)
  - 20 named ranges

The stress phase fires concurrent `retrieve_parents()` calls on the full pool of
ontology IDs (channels + tasks), replicating the RBAC / status-update traversal
pressure that the server experiences during a task restart storm.  At customer
scale this is enough to cause visible RSS growth when the server's GC is unable
to keep pace — achieved here with GOGC=2000 / GOMAXPROCS=3.

Recommended server invocation (from repo root):
    GOGC=2000 GOMAXPROCS=3 go run -tags driver core/main.go start -i --debug

Run from client/py/:
    uv run python examples/reproduce_mem_spike/3_lifelike_storm.py

Env vars:
    WORKERS=50         concurrent traversal workers (default 50)
    DURATION=120       test duration in seconds (default 120)
    BURST=1            synchronised burst waves via Barrier (default 1; 0 = steady)
    BURST_GAP_MS=50    ms of rest between burst waves (default 50)
    SKIP_BUILD=0       set to 1 to skip ontology build and reuse existing lifelike_ entities
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
from examples.opcua.server import OPCUASim
from synnax.ontology.payload import ID

# ─────────────────────────────────────────────────────────────────────────────
# Configuration — tuned to match customer environment
# ─────────────────────────────────────────────────────────────────────────────

# Ontology scale (matches ~200-channel customer report)
N_INDEX_CHANNELS = 7
N_DATA_CHANNELS_PER_INDEX = 24   # 7 × 24 = 168 raw channels
N_CALC_CHANNELS = 30             # 30 calculated channels (separate index)
N_OPC_TASKS = 2                  # 2 OPC UA tasks (driver-validated, like customer)
N_FAKE_NI_TASKS = 3              # 3 NI records via tasks.create() — no driver needed on macOS
N_RANGES = 20
PREFIX = "lifelike_"

# OPC UA float node IDs (from examples/opcua/server.py, namespace 2)
OPC_FLOAT_NODES = [
    ("NS=2;I=8",  "my_float_0"),
    ("NS=2;I=9",  "my_float_1"),
    ("NS=2;I=10", "my_float_2"),
]

# Stress configuration
WORKERS = int(os.environ.get("WORKERS", 50))
DURATION = int(os.environ.get("DURATION", 120))
BURST_MODE = os.environ.get("BURST", "1") == "1"
BURST_GAP_MS = int(os.environ.get("BURST_GAP_MS", 50))
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
# Phase 1 — Build lifelike ontology
# ─────────────────────────────────────────────────────────────────────────────


def build_ontology(client: sy.Synnax) -> list[ID]:
    """Create a small customer-scale ontology and return all ontology IDs."""

    print("=" * 70)
    print("Phase 1 — Building lifelike ontology")
    print("=" * 70)

    # ── Clean up old lifelike_ tasks ─────────────────────────────────────────
    print("\nCleaning up old lifelike_ tasks...")
    existing = client.tasks.list()
    old_tasks = [t for t in existing if t.name.startswith(PREFIX)]
    if old_tasks:
        client.tasks.delete([t.key for t in old_tasks])
        print(f"  ✓ Deleted {len(old_tasks)} old tasks")
    else:
        print("  ✓ Nothing to clean up")

    # ── Start OPC UA sim ─────────────────────────────────────────────────────
    print("\nStarting OPC UA simulator on port 4841...")
    sim = OPCUASim()
    sim.start()
    print(f"  ✓ Simulator ready at {OPCUASim.endpoint}")

    # ── Register OPC UA device ───────────────────────────────────────────────
    print("\nRegistering OPC UA device...")
    rack = client.racks.retrieve_embedded_rack()
    existing_dev = client.devices.retrieve(
        name=OPCUASim.device_name, ignore_not_found=True
    )
    if existing_dev is not None:
        opc_device = existing_dev
        print(f"  ✓ Device already registered (key={opc_device.key})")
    else:
        device = sy.opcua.Device(
            endpoint=OPCUASim.endpoint,
            name=OPCUASim.device_name,
            location=OPCUASim.endpoint,
            rack=rack.key,
        )
        opc_device = client.devices.create(device)
        print(f"  ✓ Device registered (key={opc_device.key})")

    # ── Raw index + data channels ────────────────────────────────────────────
    print(f"\nCreating {N_INDEX_CHANNELS} index channels + "
          f"{N_INDEX_CHANNELS * N_DATA_CHANNELS_PER_INDEX} data channels...")
    index_channels: list[sy.Channel] = []
    for i in range(N_INDEX_CHANNELS):
        ch = client.channels.create(
            name=f"{PREFIX}time_{i}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        index_channels.append(ch)

    data_channels = client.channels.create(
        [
            sy.Channel(
                name=f"{PREFIX}data_{i}_{j}",
                data_type=sy.DataType.FLOAT32,
                index=index_channels[i].key,
            )
            for i in range(N_INDEX_CHANNELS)
            for j in range(N_DATA_CHANNELS_PER_INDEX)
        ],
        retrieve_if_name_exists=True,
    )
    print(f"  ✓ {len(data_channels)} data channels")

    # ── Calculated channels ──────────────────────────────────────────────────
    print(f"\nCreating {N_CALC_CHANNELS} calculated channels...")
    calc_time = client.channels.create(
        name=f"{PREFIX}calc_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )
    calc_channels = client.channels.create(
        [
            sy.Channel(
                name=f"{PREFIX}calc_{k}",
                data_type=sy.DataType.FLOAT32,
                index=calc_time.key,
            )
            for k in range(N_CALC_CHANNELS)
        ],
        retrieve_if_name_exists=True,
    )
    print(f"  ✓ {len(calc_channels)} calculated channels")

    # ── OPC index channel ────────────────────────────────────────────────────
    opc_time = client.channels.create(
        name=f"{PREFIX}opc_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )

    # ── OPC UA tasks (driver-validated) ─────────────────────────────────────
    print(f"\nConfiguring {N_OPC_TASKS} OPC UA tasks (driver ACK required)...")
    opc_tasks: list[sy.task.Task] = []
    for i in range(N_OPC_TASKS):
        task_channels = client.channels.create(
            [
                sy.Channel(
                    name=f"{PREFIX}opc_{i}_{node_name}",
                    data_type=sy.DataType.FLOAT32,
                    index=opc_time.key,
                )
                for _, node_name in OPC_FLOAT_NODES
            ],
            retrieve_if_name_exists=True,
        )
        tsk = sy.opcua.ReadTask(
            name=f"{PREFIX}opc_task_{i}",
            device=opc_device.key,
            sample_rate=sy.Rate.HZ * 10,
            stream_rate=sy.Rate.HZ * 10,
            data_saving=False,
            channels=[
                sy.opcua.ReadChannel(
                    channel=task_channels[j].key,
                    node_id=node_id,
                    data_type="float32",
                )
                for j, (node_id, _) in enumerate(OPC_FLOAT_NODES)
            ],
        )
        try:
            client.tasks.configure(tsk)
            opc_tasks.append(tsk)
            print(f"  [{i + 1}/{N_OPC_TASKS}] {tsk.name} ✓")
        except Exception as e:
            print(f"  [{i + 1}/{N_OPC_TASKS}] {tsk.name} FAILED: {e}")
    print(f"  ✓ {len(opc_tasks)}/{N_OPC_TASKS} OPC tasks configured")

    # ── Fake NI tasks (raw creates — no driver needed) ───────────────────────
    # Creates task records in the ontology without driver validation.
    # Simulates the NI tasks in the customer's setup on macOS where NI-DAQmx
    # is unavailable. These exist as ontology nodes and are traversed by
    # retrieve_parents() just like real tasks.
    print(f"\nCreating {N_FAKE_NI_TASKS} NI-analog-read task records (no driver)...")
    fake_ni_tasks: list[sy.task.Task] = []
    for i in range(N_FAKE_NI_TASKS):
        tsk = client.tasks.create(
            name=f"{PREFIX}ni_task_{i}",
            type="ni_analog_read",
            config={},
        )
        fake_ni_tasks.append(tsk)
        print(f"  [{i + 1}/{N_FAKE_NI_TASKS}] {tsk.name} ✓")

    all_tasks = opc_tasks + fake_ni_tasks

    # ── Named ranges ─────────────────────────────────────────────────────────
    print(f"\nCreating {N_RANGES} named ranges...")
    now = sy.TimeStamp.now()
    for i in range(N_RANGES):
        client.ranges.create(
            name=f"{PREFIX}range_{i}",
            time_range=sy.TimeRange(
                now + i * sy.TimeSpan.SECOND,
                now + (i + 1) * sy.TimeSpan.SECOND,
            ),
        )
    print(f"  ✓ {N_RANGES} ranges created")

    # ── Build ontology ID pool ────────────────────────────────────────────────
    # Channels + tasks will all be queried in the stress phase.
    all_channel_ids = (
        [ID(key=str(int(ch.key)), type="channel") for ch in index_channels]
        + [ID(key=str(int(ch.key)), type="channel") for ch in data_channels]
        + [ID(key=str(int(calc_time.key)), type="channel")]
        + [ID(key=str(int(ch.key)), type="channel") for ch in calc_channels]
        + [ID(key=str(int(opc_time.key)), type="channel")]
    )
    all_task_ids = [ID(key=str(int(t.key)), type="task") for t in all_tasks]
    all_ids = all_channel_ids + all_task_ids

    total_ch = (
        N_INDEX_CHANNELS
        + len(data_channels)
        + 1  # calc_time
        + len(calc_channels)
        + 1  # opc_time
        + N_OPC_TASKS * len(OPC_FLOAT_NODES)
    )

    print("\n" + "=" * 70)
    print("Ontology build complete")
    print("=" * 70)
    print(f"  Channels : {total_ch} total")
    print(f"  Tasks    : {len(all_tasks)} ({len(opc_tasks)} OPC validated + "
          f"{len(fake_ni_tasks)} NI raw records)")
    print(f"  Ranges   : {N_RANGES}")
    print(f"  IDs in traversal pool: {len(all_ids)}")
    return all_ids


def load_existing_ids(client: sy.Synnax) -> list[ID]:
    """Reload ontology IDs from existing lifelike_ entities without rebuilding."""
    print("SKIP_BUILD=1 — loading existing lifelike_ entities...")
    ch_names = (
        [f"{PREFIX}time_{i}" for i in range(N_INDEX_CHANNELS)]
        + [
            f"{PREFIX}data_{i}_{j}"
            for i in range(N_INDEX_CHANNELS)
            for j in range(N_DATA_CHANNELS_PER_INDEX)
        ]
        + [f"{PREFIX}calc_time"]
        + [f"{PREFIX}calc_{k}" for k in range(N_CALC_CHANNELS)]
        + [f"{PREFIX}opc_time"]
    )
    channels = client.channels.retrieve(ch_names)
    ch_ids = [ID(key=str(int(c.key)), type="channel") for c in channels]

    tasks = [t for t in client.tasks.list() if t.name.startswith(PREFIX)]
    task_ids = [ID(key=str(int(t.key)), type="task") for t in tasks]

    ids = ch_ids + task_ids
    print(f"  ✓ {len(ch_ids)} channels + {len(task_ids)} tasks = {len(ids)} total IDs")
    return ids


# ─────────────────────────────────────────────────────────────────────────────
# Phase 2 — Stress traversal
# ─────────────────────────────────────────────────────────────────────────────

_counter_lock = threading.Lock()
_total_calls = 0
_stop_event = threading.Event()


def _increment(n: int) -> None:
    global _total_calls
    with _counter_lock:
        _total_calls += n


def worker(worker_id: int, ids: list[ID], barrier: threading.Barrier | None) -> int:
    wc = sy.Synnax(host=HOST, port=PORT, username="synnax", password="seldon")
    calls = 0
    idx = worker_id % len(ids)
    try:
        while not _stop_event.is_set():
            if barrier is not None:
                try:
                    barrier.wait(timeout=15.0)
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


def print_stats(
    elapsed: float,
    last_calls: int,
    server_pid: int | None,
    baseline_rss: int | None,
) -> int:
    global _total_calls
    with _counter_lock:
        current = _total_calls
    delta = current - last_calls
    calls_per_sec = delta / 5.0
    rss = _rss_mb(server_pid)
    rss_delta = f" (+{rss - baseline_rss:+d} MB)" if (rss and baseline_rss) else ""
    rss_str = f"  RSS={rss}{rss_delta}" if rss else ""
    print(
        f"  [{elapsed:5.0f}s] "
        f"{calls_per_sec:6.1f} calls/s"
        f"{rss_str} | "
        f"{current} total calls"
    )
    return current


def stress_traversal(ids: list[ID]) -> None:
    server_pid = _find_server_pid()
    baseline_rss = _rss_mb(server_pid)

    mode_str = (
        f"BURST (gap={BURST_GAP_MS} ms)" if BURST_MODE else "steady-state"
    )
    barrier = threading.Barrier(WORKERS) if BURST_MODE else None

    print("\n" + "=" * 70)
    print("Phase 2 — Stress traversal")
    print("=" * 70)
    print(f"  Mode     : {mode_str}")
    print(f"  Workers  : {WORKERS}")
    print(f"  Duration : {DURATION}s")
    print(f"  IDs pool : {len(ids)} ontology IDs")
    if baseline_rss:
        print(f"  Server   : PID={server_pid}  baseline RSS={baseline_rss} MB")
    print()
    print("TIP: start server with GOGC=2000 GOMAXPROCS=3 for visible RSS growth")
    print()

    start_time = time.time()
    last_calls = 0

    executor = ThreadPoolExecutor(max_workers=WORKERS)
    futures = [executor.submit(worker, wid, ids, barrier) for wid in range(WORKERS)]
    print(f"Starting {WORKERS} workers... stats every 5s.\n")

    try:
        while time.time() - start_time < DURATION:
            time.sleep(5)
            elapsed = time.time() - start_time
            last_calls = print_stats(elapsed, last_calls, server_pid, baseline_rss)
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
        "\nTo capture allocs profile during a future run (separate terminal):\n"
        f"  curl http://{HOST}:{PORT}/debug/pprof/allocs > allocs_lifelike.bin\n"
        "  go tool pprof -text -sample_index=alloc_space allocs_lifelike.bin | head -15"
    )


# ─────────────────────────────────────────────────────────────────────────────
# Entry point
# ─────────────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    client = sy.Synnax(
        host=HOST, port=PORT, username="synnax", password="seldon"
    )

    if SKIP_BUILD:
        ontology_ids = load_existing_ids(client)
    else:
        ontology_ids = build_ontology(client)

    stress_traversal(ontology_ids)
