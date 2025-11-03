#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Simulated DAQ - Creates simulated channels from task configs and writes realistic data.

This script reads Synnax task configuration JSONs and creates corresponding channels
with simulated sensor data. It's designed to help debug control systems without
requiring real hardware.
"""

import argparse
import json
import math
import random
import signal
import sys
from pathlib import Path
from typing import Any

import synnax as sy


class SimDAQ:
    def __init__(self, client: sy.Synnax, config_dir: Path):
        self.client = client
        self.running = True
        self.config_dir = config_dir

        # Channel tracking
        self.sensor_channels: list[tuple[str, str]] = []  # (name, type)
        self.valve_pairs: list[tuple[str, str]] = []  # (cmd_name, state_name)
        self.ao_pairs: list[tuple[str, str]] = []  # (cmd_name, state_name)

        # Simulation state
        self.valve_states: dict[str, float] = {}
        self.ao_states: dict[str, float] = {}
        self.time_offset = sy.TimeStamp.now()
        self.noise = random.Random()

        # Index channels
        self.indices: dict[str, sy.Channel] = {}

        # Channel mapping: config_key -> {name, synnax_key}
        self.channel_mapping: list[dict[str, Any]] = []

    def get_index(self, name: str) -> sy.Channel:
        """Get or create an index channel."""
        if name not in self.indices:
            self.indices[name] = self.client.channels.create(
                name=name,
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
                retrieve_if_name_exists=True,
            )
        return self.indices[name]

    def setup_channels(self) -> None:
        print("=" * 70)
        print("CREATING SIMULATED CHANNELS")
        print("=" * 70)

        json_files = sorted(self.config_dir.glob("*.json"))
        print(f"Found {len(json_files)} config files\n")

        for json_file in json_files:
            with open(json_file) as f:
                config = json.load(f)

            # Handle two formats: {"channels": [...]} or just [...]
            if isinstance(config, list):
                channels = config
            elif isinstance(config, dict):
                channels = config.get("channels", [])
            else:
                print(f"[{json_file.name}] ⚠ Skipping - unknown format")
                continue

            if not channels:
                continue

            first = channels[0]

            # Digital/analog outputs with cmd/state pairs
            if "cmdChannel" in first and "stateChannel" in first:
                if first.get("type") == "digital_output":
                    self._create_digital_outputs(channels, json_file.name)
                elif first.get("type", "").startswith("ao_"):
                    self._create_analog_outputs(channels, json_file.name)
            # Analog inputs (sensors)
            elif "channel" in first:
                self._create_analog_inputs(channels, json_file.name)

        print(f"\n{'=' * 70}")
        print(f"SETUP COMPLETE")
        print(f"  Indices:         {len(self.indices)}")
        print(f"  Sensors:         {len(self.sensor_channels)}")
        print(f"  Valve pairs:     {len(self.valve_pairs)}")
        print(f"  Analog outputs:  {len(self.ao_pairs)}")
        print(f"{'=' * 70}\n")

        # Save channel mapping
        self._save_channel_mapping()

    def _create_digital_outputs(
        self, channels: list[dict[str, Any]], filename: str
    ) -> None:
        """Create digital output channels (valves)."""
        print(f"[{filename}] Digital outputs")

        state_idx = self.get_index("daq_time")

        for i, ch in enumerate(channels):
            if not ch.get("enabled", True):
                continue

            line = ch.get("line", 0)
            config_cmd_key = ch.get("cmdChannel")
            config_state_key = ch.get("stateChannel")

            # Create simple sequential names like valve_1_cmd, valve_1_state
            cmd_name = f"valve_{line}_cmd"
            state_name = f"valve_{line}_state"

            # Each command channel gets its own unique index
            cmd_idx = self.get_index(f"{cmd_name}_time")

            cmd_ch = self.client.channels.create(
                name=cmd_name,
                data_type=sy.DataType.UINT8,
                index=cmd_idx.key,
                retrieve_if_name_exists=True,
            )

            state_ch = self.client.channels.create(
                name=state_name,
                data_type=sy.DataType.UINT8,
                index=state_idx.key,
                retrieve_if_name_exists=True,
            )

            # Record mapping
            if config_cmd_key:
                self.channel_mapping.append(
                    {
                        "name": cmd_name,
                        "config_key": config_cmd_key,
                        "synnax_key": cmd_ch.key,
                    }
                )
            if config_state_key:
                self.channel_mapping.append(
                    {
                        "name": state_name,
                        "config_key": config_state_key,
                        "synnax_key": state_ch.key,
                    }
                )

            self.valve_pairs.append((cmd_name, state_name))
            self.valve_states[state_name] = 0.0
            print(f"  ✓ {cmd_name} / {state_name}")

    def _create_analog_inputs(
        self, channels: list[dict[str, Any]], filename: str
    ) -> None:
        """Create analog input channels (sensors)."""
        print(f"[{filename}] Analog inputs")

        data_idx = self.get_index("daq_time")

        for i, ch in enumerate(channels):
            if not ch.get("enabled", True):
                continue

            port = ch.get("port", 0)
            ch_type = ch.get("type", "ai")
            config_key = ch.get("channel")

            # Generate descriptive name
            name = self._generate_sensor_name(ch, port, ch_type, i)

            created_ch = self.client.channels.create(
                name=name,
                data_type=sy.DataType.FLOAT32,
                index=data_idx.key,
                retrieve_if_name_exists=True,
            )

            # Record mapping
            if config_key:
                self.channel_mapping.append(
                    {
                        "name": name,
                        "config_key": config_key,
                        "synnax_key": created_ch.key,
                    }
                )

            self.sensor_channels.append((name, ch_type))
            print(f"  ✓ {name}")

    def _create_analog_outputs(
        self, channels: list[dict[str, Any]], filename: str
    ) -> None:
        """Create analog output channels."""
        print(f"[{filename}] Analog outputs")

        state_idx = self.get_index("daq_time")

        for i, ch in enumerate(channels):
            if not ch.get("enabled", True):
                continue

            port = ch.get("port", 0)
            config_cmd_key = ch.get("cmdChannel")
            config_state_key = ch.get("stateChannel")

            # Simple sequential naming
            cmd_name = f"ao_{port}_cmd"
            state_name = f"ao_{port}_state"

            # Each command channel gets its own unique index
            cmd_idx = self.get_index(f"{cmd_name}_time")

            cmd_ch = self.client.channels.create(
                name=cmd_name,
                data_type=sy.DataType.FLOAT32,
                index=cmd_idx.key,
                retrieve_if_name_exists=True,
            )

            state_ch = self.client.channels.create(
                name=state_name,
                data_type=sy.DataType.FLOAT32,
                index=state_idx.key,
                retrieve_if_name_exists=True,
            )

            # Record mapping
            if config_cmd_key:
                self.channel_mapping.append(
                    {
                        "name": cmd_name,
                        "config_key": config_cmd_key,
                        "synnax_key": cmd_ch.key,
                    }
                )
            if config_state_key:
                self.channel_mapping.append(
                    {
                        "name": state_name,
                        "config_key": config_state_key,
                        "synnax_key": state_ch.key,
                    }
                )

            self.ao_pairs.append((cmd_name, state_name))
            self.ao_states[state_name] = 0.0
            print(f"  ✓ {cmd_name} / {state_name}")

    def _generate_sensor_name(
        self, ch: dict[str, Any], port: int, ch_type: str, index: int
    ) -> str:
        """Generate a descriptive sensor name."""
        if ch_type == "ai_voltage":
            return f"voltage_{port}"

        elif ch_type == "ai_current":
            return f"current_{port}"

        elif ch_type == "ai_bridge":
            return f"bridge_{port}"

        elif ch_type == "ai_thermocouple":
            tc_type = ch.get("thermocoupleType", "K")
            return f"temp_tc{tc_type}_{port}"

        elif ch_type == "ai_rtd":
            return f"temp_rtd_{port}"
        else:
            return f"sensor_{index}"

    def _save_channel_mapping(self) -> None:
        """Save channel mapping to JSON file."""
        mapping_file = self.config_dir / "channel_mapping.json"
        with open(mapping_file, "w") as f:
            json.dump(self.channel_mapping, f, indent=2)
        print(f"✓ Saved channel mapping to: {mapping_file}")
        print(f"  Total mappings: {len(self.channel_mapping)}\n")

    def generate_sensor_value(
        self, name: str, ch_type: str, timestamp: sy.TimeStamp
    ) -> float:
        """Generate realistic sensor data."""
        t = (timestamp - self.time_offset) / sy.TimeSpan.SECOND
        freq = 0.1 + (hash(name) % 10) * 0.05
        base = math.sin(2 * math.pi * freq * t)
        noise = self.noise.uniform(-0.05, 0.05)

        name_lower = name.lower()

        # Pressure sensors
        if "press" in name_lower or "psi" in name_lower:
            return 50 + base * 30 + noise * 5

        # Temperature sensors
        if "temp" in name_lower or ch_type in ["ai_thermocouple", "ai_rtd"]:
            return 70 + base * 10 + noise * 2

        # Voltage sensors
        if "voltage" in name_lower:
            if "24" in name_lower:
                return 12 + base * 6 + noise
            elif "5" in name_lower:
                return 2.5 + base * 1.5 + noise * 0.5
            else:
                return 0.015 + base * 0.01 + noise * 0.001

        # Current sensors
        if "current" in name_lower or ch_type == "ai_current":
            return 0.012 + base * 0.004 + noise * 0.0005

        # Load cells
        if "load" in name_lower or "lb" in name_lower or ch_type == "ai_bridge":
            return base * 500 + noise * 50

        # Flow
        if "flow" in name_lower:
            return 50 + base * 30 + noise * 5

        # Percentage
        if "pct" in name_lower or "%" in name_lower:
            return 50 + base * 30 + noise * 5

        # Default
        return base * 10 + noise

    def run(self) -> None:
        print("=" * 70)
        print("STARTING SIMULATED DAQ")
        print("=" * 70)

        # Setup signal handler
        def stop(s: int, f: Any) -> None:
            self.running = False

        signal.signal(signal.SIGINT, stop)
        signal.signal(signal.SIGTERM, stop)

        # Build channel lists
        sensor_names = [name for name, _ in self.sensor_channels]
        cmd_channels = [cmd for cmd, _ in self.valve_pairs] + [
            cmd for cmd, _ in self.ao_pairs
        ]
        state_channels = [state for _, state in self.valve_pairs] + [
            state for _, state in self.ao_pairs
        ]
        write_channels = ["daq_time"] + sensor_names + state_channels

        print(f"\nConfiguration:")
        print(f"  Writing {len(write_channels)} channels at 100 Hz")
        print(f"  Listening to {len(cmd_channels)} command channels")

        # Open streamer for commands
        streamer = None
        if cmd_channels:
            try:
                streamer = self.client.open_streamer(cmd_channels)
                print(f"  ✓ Command streamer opened")
            except Exception as e:
                print(f"  ⚠ Warning: {e}")

        # Open writer for data
        writer = self.client.open_writer(
            start=sy.TimeStamp.now(),
            channels=write_channels,
            name="CTS Sim DAQ",
            enable_auto_commit=True,
        )
        print(f"  ✓ Data writer opened")

        loop = sy.Loop(sy.Rate.HZ * 100)
        iteration = 0
        last_status = sy.TimeStamp.now()

        print(f"\n{'=' * 70}")
        print("RUNNING (Ctrl+C to stop)")
        print(f"{'=' * 70}\n")

        try:
            while self.running and loop.wait():
                timestamp = sy.TimeStamp.now()

                # Read commands
                if streamer:
                    try:
                        frame = streamer.read(timeout=0)
                        if frame is not None:
                            # Process valve commands
                            for cmd_name, state_name in self.valve_pairs:
                                data = frame.get(cmd_name, [])
                                if len(data) > 0:
                                    self.valve_states[state_name] = float(data[-1])
                                    if data[-1] != 0:
                                        print(
                                            f"  [{timestamp}] {cmd_name} = {data[-1]}"
                                        )

                            # Process analog output commands
                            for cmd_name, state_name in self.ao_pairs:
                                data = frame.get(cmd_name, [])
                                if len(data) > 0:
                                    target = float(data[-1])
                                    current = self.ao_states.get(state_name, 0.0)
                                    self.ao_states[state_name] = current + 0.8 * (
                                        target - current
                                    )
                                    if abs(target) > 0.001:
                                        print(
                                            f"  [{timestamp}] {cmd_name} = {target:.4f}"
                                        )
                    except Exception as e:
                        if iteration % 1000 == 0:
                            print(f"  ⚠ Error: {e}")

                # Build frame
                frame_data = {"daq_time": timestamp}

                # Add sensor data
                for name, ch_type in self.sensor_channels:
                    frame_data[name] = self.generate_sensor_value(
                        name, ch_type, timestamp
                    )

                # Add valve states
                for _, state_name in self.valve_pairs:
                    frame_data[state_name] = self.valve_states.get(state_name, 0.0)

                # Add analog output states
                for _, state_name in self.ao_pairs:
                    frame_data[state_name] = self.ao_states.get(state_name, 0.0)

                # Write
                writer.write(frame_data)
                iteration += 1

                # Status update every 10 seconds
                if (timestamp - last_status) > sy.TimeSpan.SECOND * 10:
                    print(
                        f"  [{timestamp}] {iteration} frames, {len(frame_data)} channels"
                    )
                    last_status = timestamp

        except KeyboardInterrupt:
            print("\n\nKeyboard interrupt")
        except Exception as e:
            print(f"\n\nERROR: {e}")
            import traceback

            traceback.print_exc()
        finally:
            print(f"\n{'=' * 70}")
            print("SHUTTING DOWN")
            print(f"{'=' * 70}")
            writer.close()
            if streamer:
                streamer.close()
            print(f"✓ Stopped after {iteration} iterations\n")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Simulated DAQ - Creates channels from Synnax task configs and writes simulated data.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s /path/to/configs/
  %(prog)s ./task_configs/

This script reads Synnax task configuration JSON files and creates corresponding
channels with simulated sensor data. It's designed to help debug control systems
without requiring real hardware.

The script will:
  1. Read all *.json files in the specified directory
  2. Create channels in Synnax based on the task configs
  3. Save a channel mapping file (config_key → synnax_key)
  4. Start writing simulated data at 100 Hz
  5. Listen for commands and update states accordingly
        """,
    )
    parser.add_argument(
        "config_dir",
        type=str,
        help="Path to directory containing Synnax task configuration JSON files",
    )

    args = parser.parse_args()
    config_dir = Path(args.config_dir)

    # Validate directory
    if not config_dir.exists():
        print(f"✗ ERROR: Directory does not exist: {config_dir}\n")
        sys.exit(1)

    if not config_dir.is_dir():
        print(f"✗ ERROR: Not a directory: {config_dir}\n")
        sys.exit(1)

    # Check for JSON files
    json_files = list(config_dir.glob("*.json"))
    if not json_files:
        print(f"✗ ERROR: No JSON files found in: {config_dir}\n")
        sys.exit(1)

    print(f"\n{'=' * 70}")
    print("SIMULATED DAQ")
    print(f"{'=' * 70}\n")
    print(f"Config directory: {config_dir.absolute()}")
    print(f"Found {len(json_files)} config files\n")

    print("Connecting to Synnax...")
    try:
        client = sy.Synnax()
        print("✓ Connected\n")
    except Exception as e:
        print(f"✗ ERROR: {e}\n")
        sys.exit(1)

    daq = SimDAQ(client, config_dir)
    daq.setup_channels()
    daq.run()


if __name__ == "__main__":
    main()
