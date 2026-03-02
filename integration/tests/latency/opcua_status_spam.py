#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Memory spike stress test — driver StatusService.Set spam.

Scenario:
    - OPC UA (plain + TLS) + Modbus sims via SimulatorCase
    - N_TASKS OPC UA array-mode read tasks with array_size mismatch (10 vs 5)
    - OPC UA write task + Modbus read/write tasks (normal operation)
    - TLS OPC UA read tasks (additional warning generators)
    - Background writer pushing uptime data to bulk channels
    - Every array mismatch fires send_warning() -> StatusService.Set -> RBAC
    - N_TASKS x SAMPLE_RATE ~ total StatusService.Set calls/s from warnings

Setup sequence:
    1.  [Claude] rm -rf synnax-data/
    2.  [Claude] GOGC=3000 GOMAXPROCS=2 go run -tags driver core/main.go start -i
    3.  [Script] Auto-start OPC UA (plain + TLS) + Modbus simulators
    4.  [Script] Build bulk channels + calc + ranges + fake NI tasks
    5.  [Script] Configure all driver tasks (spam + normal)
    6.  [Script] Open persistent writer to bulk channels
    7.  [Script] Monitor RSS every 1s for DURATION seconds

Run from integration/:
    uv run python -m tests.latency.opcua_status_spam

Env vars:
    DURATION=300        Total test duration in seconds
    N_TASKS=3           OPC UA array-mode read tasks per sim (warning generators)
    SAMPLE_RATE=200     Per-task sample rate in Hz
    N_INDEX=4           Index channels (x N_DATA = bulk channel count)
    N_DATA=50           Data channels per index (default 50 = 1000 total)
    N_CALC=50           Calc channel count
    N_RANGES=50         Range count
    N_FAKE_NI=5         Fake NI task records (ontology weight)
    SKIP_BUILD=0        Set 1 to reuse existing entities
    SYNNAX_HOST         Server host (default localhost)
    SYNNAX_PORT         Server port (default 9090)
"""

import math
import os
import subprocess
import time
from datetime import datetime, timedelta, timezone

import numpy as np
import synnax as sy
from examples.modbus import ModbusSim
from examples.opcua import OPCUASim, OPCUATLSSim

from framework.test_case import SynnaxConnection
from tests.driver.simulator_case import SimulatorCase

PREFIX = "status_spam_"

DURATION = int(os.environ.get("DURATION", 300))
N_TASKS = int(os.environ.get("N_TASKS", 3))
SAMPLE_RATE = int(os.environ.get("SAMPLE_RATE", 1000))
N_INDEX = int(os.environ.get("N_INDEX", 4))
N_DATA = int(os.environ.get("N_DATA", 50))
N_CALC = int(os.environ.get("N_CALC", 50))
N_RANGES = int(os.environ.get("N_RANGES", 50))
N_FAKE_NI = int(os.environ.get("N_FAKE_NI", 5))
SKIP_BUILD = os.environ.get("SKIP_BUILD", "0") == "1"
HOST = os.environ.get("SYNNAX_HOST", "localhost")
PORT = int(os.environ.get("SYNNAX_PORT", 9090))

# OPC UA array nodes (namespace 2, sim serves size=5 each)
OPC_ARRAY_NODES = [
    ("NS=2;I=2", "my_array_0"),
    ("NS=2;I=3", "my_array_1"),
    ("NS=2;I=4", "my_array_2"),
    ("NS=2;I=5", "my_array_3"),
    ("NS=2;I=6", "my_array_4"),
]

# OPC UA command nodes (writable)
OPC_CMD_NODES = [
    ("NS=2;I=18", "command_0"),
    ("NS=2;I=19", "command_1"),
    ("NS=2;I=20", "command_2"),
]

# Mismatch size: sim serves 5, task expects 2 -> warning every read
# Read cycle rate = sample_rate / array_size, so smaller = more cycles/s
MISMATCH_ARRAY_SIZE = 2


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


def _total_ram_mb() -> int:
    """Get total physical RAM in MB."""
    try:
        r = subprocess.run(
            ["sysctl", "-n", "hw.memsize"],
            capture_output=True, text=True, timeout=2,
        )
        return int(r.stdout.strip()) // (1024 * 1024)
    except Exception:
        return 0


# ─────────────────────────────────────────────────────────────────────────────
# Test case
# ─────────────────────────────────────────────────────────────────────────────


class DriverStatusSpam(SimulatorCase):
    """Memory spike stress test via driver StatusService.Set spam.

    Launches OPC UA (plain + TLS) + Modbus simulators, configures:
    - N_TASKS OPC UA array-mode read tasks per sim (mismatch -> warnings)
    - 1 OPC UA write task (normal)
    - 1 Modbus read task + 1 Modbus write task (normal)
    - Background writer pushing uptime to bulk channels
    - 1k bulk channels + 50 calc + 50 ranges + 5 fake NI tasks (ontology)
    """

    sim_classes = [OPCUASim, OPCUATLSSim, ModbusSim]
    SAMPLE_RATE = sy.Rate.HZ * max(SAMPLE_RATE, 1000)

    def setup(self) -> None:
        super().setup()
        self._index_chs: list[sy.Channel] = []
        self._data_chs: list[sy.Channel] = []
        self._build_entities()

    def _build_entities(self) -> None:
        if SKIP_BUILD:
            self.log("SKIP_BUILD=1 -- reusing existing entities")
            return

        t0 = time.time()
        self.log(
            f"Building ontology: {N_INDEX * N_DATA} data + {N_CALC} calc "
            f"channels, {N_RANGES} ranges, {N_FAKE_NI} tasks"
        )

        # Clean up old tasks
        existing_tasks = [
            t for t in self.client.tasks.list() if t.name.startswith(PREFIX)
        ]
        if existing_tasks:
            self.client.tasks.delete([t.key for t in existing_tasks])

        # Index channels
        self._index_chs = self.client.channels.create(
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
        self._data_chs = self.client.channels.create(
            [
                sy.Channel(
                    name=f"{PREFIX}data_{i}_{j}",
                    data_type=sy.DataType.FLOAT32,
                    index=self._index_chs[i].key,
                )
                for i in range(N_INDEX)
                for j in range(N_DATA)
            ],
            retrieve_if_name_exists=True,
        )

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
            self.client.ranges.create(
                name=f"{PREFIX}range_{r}",
                time_range=sy.TimeRange(
                    int(start_dt.timestamp() * 1e9),
                    int(end_dt.timestamp() * 1e9),
                ),
                retrieve_if_name_exists=True,
            )

        # Fake NI tasks (ontology weight) — created last, named to sort after all others
        for i in range(N_FAKE_NI):
            self.client.tasks.create(
                sy.Task(
                    name=f"{PREFIX}zz_ni_task_{i}",
                    type="ni_analog_read",
                    config={},
                )
            )

        elapsed = time.time() - t0
        self.log(f"  Ontology built in {elapsed:.1f}s")

    def _configure_opc_spam_tasks(
        self, device_name: str, tag: str
    ) -> list[sy.Task]:
        """OPC UA array-mode read tasks with size mismatch -> warning spam."""
        dev = self.client.devices.retrieve(name=device_name)

        opc_time = self.client.channels.create(
            name=f"{PREFIX}{tag}_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )

        tasks: list[sy.Task] = []
        for i in range(N_TASKS):
            task_channels = self.client.channels.create(
                [
                    sy.Channel(
                        name=f"{PREFIX}{tag}_{i}_{node_name}",
                        data_type=sy.DataType.FLOAT32,
                        index=opc_time.key,
                    )
                    for _, node_name in OPC_ARRAY_NODES
                ],
                retrieve_if_name_exists=True,
            )

            tsk = sy.opcua.ReadTask(
                name=f"{PREFIX}{tag}_spam_{i}",
                device=dev.key,
                sample_rate=sy.Rate.HZ * SAMPLE_RATE,
                array_mode=True,
                array_size=MISMATCH_ARRAY_SIZE,
                data_saving=False,
                auto_start=True,
                channels=[
                    sy.opcua.ReadChannel(
                        channel=task_channels[j].key,
                        node_id=node_id,
                        data_type="float32",
                    )
                    for j, (node_id, _) in enumerate(OPC_ARRAY_NODES)
                ],
            )
            try:
                self.client.tasks.configure(tsk)
                tasks.append(tsk)
                self.log(f"  [{i + 1}/{N_TASKS}] {tsk.name} started")
            except Exception as e:
                self.log(f"  [{i + 1}/{N_TASKS}] {tsk.name} FAILED: {e}")

        return tasks

    def _configure_opc_write_task(self) -> list[sy.Task]:
        """OPC UA write task (normal operation)."""
        opc_dev = self.client.devices.retrieve(name=OPCUASim.device_name)

        tasks: list[sy.Task] = []

        cmd_time = self.client.channels.create(
            name=f"{PREFIX}opc_cmd_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        cmd_chs = self.client.channels.create(
            [
                sy.Channel(
                    name=f"{PREFIX}opc_cmd_{name}",
                    data_type=sy.DataType.FLOAT32,
                    index=cmd_time.key,
                )
                for _, name in OPC_CMD_NODES
            ],
            retrieve_if_name_exists=True,
        )

        write_tsk = sy.opcua.WriteTask(
            name=f"{PREFIX}opc_write",
            device=opc_dev.key,
            auto_start=True,
            channels=[
                sy.opcua.WriteChannel(
                    cmd_channel=cmd_chs[j].key,
                    node_id=node_id,
                )
                for j, (node_id, _) in enumerate(OPC_CMD_NODES)
            ],
        )
        try:
            self.client.tasks.configure(write_tsk)
            tasks.append(write_tsk)
            self.log(f"  {write_tsk.name} started")
        except Exception as e:
            self.log(f"  {write_tsk.name} FAILED: {e}")

        return tasks

    def _configure_modbus_tasks(self) -> list[sy.Task]:
        """Modbus read + write tasks (normal operation)."""
        mb_dev = self.client.devices.retrieve(name=ModbusSim.device_name)

        mb_time = self.client.channels.create(
            name=f"{PREFIX}mb_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )

        tasks: list[sy.Task] = []

        # Read task — 2 input registers
        mb_read_chs = self.client.channels.create(
            [
                sy.Channel(
                    name=f"{PREFIX}mb_input_reg_{i}",
                    data_type=sy.DataType.UINT8,
                    index=mb_time.key,
                )
                for i in range(2)
            ],
            retrieve_if_name_exists=True,
        )

        mb_read = sy.modbus.ReadTask(
            name=f"{PREFIX}mb_read",
            device=mb_dev.key,
            sample_rate=sy.Rate.HZ * 10,
            stream_rate=sy.Rate.HZ * 10,
            data_saving=False,
            auto_start=True,
            channels=[
                sy.modbus.InputRegisterChan(
                    channel=mb_read_chs[i].key,
                    address=i,
                    data_type="uint8",
                )
                for i in range(2)
            ],
        )
        try:
            self.client.tasks.configure(mb_read)
            tasks.append(mb_read)
            self.log(f"  {mb_read.name} started")
        except Exception as e:
            self.log(f"  {mb_read.name} FAILED: {e}")

        # Write task — 2 coils
        mb_cmd_time = self.client.channels.create(
            name=f"{PREFIX}mb_cmd_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        mb_cmd_chs = self.client.channels.create(
            [
                sy.Channel(
                    name=f"{PREFIX}mb_coil_cmd_{i}",
                    data_type=sy.DataType.UINT8,
                    index=mb_cmd_time.key,
                )
                for i in range(2)
            ],
            retrieve_if_name_exists=True,
        )

        mb_write = sy.modbus.WriteTask(
            name=f"{PREFIX}mb_write",
            device=mb_dev.key,
            auto_start=True,
            channels=[
                sy.modbus.CoilOutputChan(
                    channel=mb_cmd_chs[i].key,
                    address=i,
                )
                for i in range(2)
            ],
        )
        try:
            self.client.tasks.configure(mb_write)
            tasks.append(mb_write)
            self.log(f"  {mb_write.name} started")
        except Exception as e:
            self.log(f"  {mb_write.name} FAILED: {e}")

        return tasks

    def run(self) -> None:
        server_pid = _find_server_pid()
        baseline_rss = _rss_mb(server_pid)

        # Subscribe to mem% channel before tasks start
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

        self.log(f"Server PID={server_pid}  baseline RSS={baseline_rss} MB")
        self.log(
            f"Duration={DURATION}s  sample_rate={SAMPLE_RATE} Hz  "
            f"data_channels={N_INDEX * N_DATA}  "
            f"calc={N_CALC}  ranges={N_RANGES}  fake_ni={N_FAKE_NI}"
        )

        # Configure all tasks
        self.log(f"Configuring {N_TASKS} plain OPC UA spam tasks")
        plain_spam = self._configure_opc_spam_tasks(
            OPCUASim.device_name, "opc"
        )
        self.log(f"Configuring {N_TASKS} encrypted OPC UA spam tasks")
        enc_spam = self._configure_opc_spam_tasks(
            OPCUATLSSim.device_name, "opc_enc"
        )
        self.log("Configuring OPC UA write task")
        opc_tasks = self._configure_opc_write_task()
        self.log("Configuring Modbus read + write tasks")
        mb_tasks = self._configure_modbus_tasks()

        spam_tasks = plain_spam + enc_spam
        all_tasks = spam_tasks + opc_tasks + mb_tasks
        expected_rate = len(spam_tasks) * (SAMPLE_RATE // MISMATCH_ARRAY_SIZE)
        self.log(
            f"Total tasks: {len(all_tasks)} "
            f"({len(plain_spam)} plain spam + {len(enc_spam)} enc spam + "
            f"{len(opc_tasks)} opc write + {len(mb_tasks)} modbus)  "
            f"~{expected_rate} StatusService.Set calls/s expected"
        )

        # Open persistent writer to bulk channels for uptime data
        write_keys = (
            [ch.key for ch in self._index_chs]
            + [ch.key for ch in self._data_chs]
        )
        writer = self.client.open_writer(
            sy.TimeStamp.now(),
            write_keys,
            name=f"{PREFIX}bulk_writer",
            enable_auto_commit=True,
        )
        self.log(
            f"Opened bulk writer on {len(write_keys)} channels "
            f"({N_INDEX} index + {N_INDEX * N_DATA} data)"
        )

        # ── Stress loop ──────────────────────────────────────────────────

        start = time.time()
        last_report = start

        self.log("")
        self.log("  [elapsed]  RSS (delta)              mem%")
        self.log("  " + "-" * 50)

        # Print baseline line
        mem_baseline_str = ""
        if baseline_mem is not None:
            mem_baseline_str = f"  mem={baseline_mem:.1f}% (+0 MB)"
        self.log(
            f"  [    0s]  RSS={baseline_rss} (+0 MB){mem_baseline_str}"
        )

        try:
            while time.time() - start < DURATION:
                now_time = time.time()
                elapsed = now_time - start

                # Write uptime sample to all bulk channels
                now_ts = sy.TimeStamp.now()
                frame = {}
                for i, idx_ch in enumerate(self._index_chs):
                    frame[idx_ch.key] = np.array(
                        [int(now_ts)], dtype=np.int64
                    )
                    for j in range(N_DATA):
                        data_idx = i * N_DATA + j
                        frame[self._data_chs[data_idx].key] = np.array(
                            [math.sin(elapsed + j)], dtype=np.float32
                        )
                writer.write(frame)

                if now_time - last_report >= 1.0:
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
                        f"RSS={rss}{rss_delta}{mem_str}"
                    )
                    last_report = now_time

                time.sleep(1)
        except KeyboardInterrupt:
            self.log("Interrupted by user")
        finally:
            writer.close()

        # ── Final summary ────────────────────────────────────────────────

        end_rss = _rss_mb(server_pid)
        growth = (
            (end_rss - baseline_rss) if (end_rss and baseline_rss) else None
        )

        self.log("")
        self.log("=" * 65)
        self.log("Driver StatusService.Set spam stress complete")
        self.log("=" * 65)
        self.log(f"  RSS start  : {baseline_rss} MB")
        self.log(f"  RSS end    : {end_rss} MB")
        if growth is not None:
            trend = "GROWING" if growth > 500 else "STABLE"
            self.log(f"  RSS growth : {growth:+d} MB  {trend}")
        self.log(
            f"  Spam tasks : {len(spam_tasks)} "
            f"({len(plain_spam)} plain + {len(enc_spam)} enc) "
            f"@ {SAMPLE_RATE} Hz"
        )
        self.log(f"  Other tasks: {len(opc_tasks) + len(mb_tasks)}")
        self.log(
            f"  Expected   : ~{expected_rate} StatusService.Set/s"
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
    test = DriverStatusSpam(
        SynnaxConnection(server_address=HOST, port=PORT),
        name="opcua_status_spam",
    )
    test.execute()
