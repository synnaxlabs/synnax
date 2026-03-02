"""
Build a large ontology to reproduce the synnax-server memory spike.

Creates channels, properly configured OPC UA tasks (using a local sim), and
ranges so that each traverseByScan call scans enough KV rows to create
significant per-call allocation pressure.

NI tasks are excluded — NI-DAQmx is not available on macOS.

Run from client/py/:
    uv run python examples/reproduce_mem_spike/1_build_ontology.py

Server must be running on localhost:9090 with the embedded driver active.
"""

import sys

import synnax as sy
from examples.opcua.server import OPCUASim

# ─────────────────────────────────────────────────────────────────────────────
# Parameters
# ─────────────────────────────────────────────────────────────────────────────

N_INDEX_CHANNELS = 50
N_DATA_CHANNELS_PER_INDEX = 100  # 50 × 100 = 5,000 data channels
N_OPC_TASKS = 20                 # properly configured via driver ack
N_RANGES = 100
PREFIX = "repro_"

# OPC UA float node IDs as created by examples/opcua/server.py
# my_float_0 → my_float_4 in namespace 2, sequential integer IDs
OPC_FLOAT_NODES = [
    ("NS=2;I=8",  "my_float_0"),
    ("NS=2;I=9",  "my_float_1"),
    ("NS=2;I=10", "my_float_2"),
    ("NS=2;I=11", "my_float_3"),
    ("NS=2;I=12", "my_float_4"),
]

# ─────────────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    client = sy.Synnax(
        host="localhost",
        port=9090,
        username="synnax",
        password="seldon",
    )

    print("=" * 70)
    print("Ontology build script")
    print("=" * 70)

    # ── 0. Clean up any previously created bad repro_ tasks ──────────────────────

    print("\nCleaning up old repro_ tasks (if any)...")
    existing_tasks = client.tasks.list()
    bad_tasks = [t for t in existing_tasks if t.name.startswith(PREFIX)]
    if bad_tasks:
        client.tasks.delete([t.key for t in bad_tasks])
        print(f"  ✓ Deleted {len(bad_tasks)} old repro_ tasks")
    else:
        print("  ✓ No old tasks to clean up")

    # ── 1. Start OPC UA simulator ─────────────────────────────────────────────────

    print("\nStarting OPC UA simulator on port 4841...")
    sim = OPCUASim()
    sim.start()
    print(f"  ✓ Simulator ready at {OPCUASim.endpoint}")

    # ── 2. Register OPC UA device ─────────────────────────────────────────────────

    print("\nRegistering OPC UA device with Synnax...")
    rack = client.racks.retrieve_embedded_rack()
    existing_device = client.devices.retrieve(
        name=OPCUASim.device_name, ignore_not_found=True
    )
    if existing_device is not None:
        opc_device = existing_device
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

    # ── 3. Create index + data channels ──────────────────────────────────────────

    print(f"\nCreating {N_INDEX_CHANNELS} index channels...")
    index_channels: list[sy.Channel] = []
    for i in range(N_INDEX_CHANNELS):
        ch = client.channels.create(
            name=f"{PREFIX}time_{i}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        index_channels.append(ch)

    total_data = N_INDEX_CHANNELS * N_DATA_CHANNELS_PER_INDEX
    print(f"Creating {total_data} data channels ({N_DATA_CHANNELS_PER_INDEX} per index)...")
    data_channel_defs = [
        sy.Channel(
            name=f"{PREFIX}data_{i}_{j}",
            data_type=sy.DataType.FLOAT32,
            index=index_channels[i].key,
        )
        for i in range(N_INDEX_CHANNELS)
        for j in range(N_DATA_CHANNELS_PER_INDEX)
    ]
    data_channels = client.channels.create(
        data_channel_defs, retrieve_if_name_exists=True
    )
    print(f"  ✓ {N_INDEX_CHANNELS + len(data_channels)} channels total")

    # ── 4. Create OPC UA tasks (driver-validated via configure()) ─────────────────

    print(f"\nConfiguring {N_OPC_TASKS} OPC read tasks (driver ack required)...")

    # Shared index channel for all OPC task output
    opc_time = client.channels.create(
        name=f"{PREFIX}opc_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )

    opc_tasks = []
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
            print(f"  [{i+1:2d}/{N_OPC_TASKS}] {tsk.name} ✓")
        except Exception as e:
            print(f"  [{i+1:2d}/{N_OPC_TASKS}] {tsk.name} FAILED: {e}", file=sys.stderr)

    print(f"  ✓ {len(opc_tasks)}/{N_OPC_TASKS} OPC tasks configured")

    # ── 5. Create ranges ──────────────────────────────────────────────────────────

    print(f"\nCreating {N_RANGES} named ranges...")
    now = sy.TimeStamp.now()
    ranges = [
        client.ranges.create(
            name=f"{PREFIX}range_{i}",
            time_range=sy.TimeRange(
                now + i * sy.TimeSpan.SECOND,
                now + (i + 1) * sy.TimeSpan.SECOND,
            ),
        )
        for i in range(N_RANGES)
    ]
    print(f"  ✓ {len(ranges)} ranges created")

    # ── Summary ───────────────────────────────────────────────────────────────────

    opc_ch_count = 1 + N_OPC_TASKS * len(OPC_FLOAT_NODES)  # opc_time + per-task floats
    total_channels = N_INDEX_CHANNELS + len(data_channels) + opc_ch_count
    total_tasks = len(opc_tasks)

    print("\n" + "=" * 70)
    print("Ontology build complete")
    print("=" * 70)
    print(f"  Channels : {total_channels}")
    print(f"  Tasks    : {total_tasks} (OPC, driver-validated)")
    print(f"  Ranges   : {N_RANGES}")
    print(
        f"\nThe OPC sim daemon will stop when this script exits — that's fine,\n"
        f"the tasks and channels remain in the ontology. Run 2_stress_traversal.py next."
    )
