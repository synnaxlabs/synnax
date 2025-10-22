#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Transform Schematic - Replaces config channel keys with Synnax keys.

This script reads a schematic JSON and channel mapping, then creates a new
schematic with config keys replaced by actual Synnax keys.
"""

import argparse
import json
import sys
from pathlib import Path
from typing import Any, Dict


def load_channel_mapping(mapping_path: Path) -> Dict[int, int]:
    """Load channel mapping and create config_key -> synnax_key dict."""
    with open(mapping_path) as f:
        mappings = json.load(f)

    # Build lookup dict: config_key -> synnax_key
    key_map = {}
    for entry in mappings:
        config_key = entry.get("config_key")
        synnax_key = entry.get("synnax_key")
        if config_key and synnax_key:
            key_map[config_key] = synnax_key

    return key_map


def transform_value(
    value: Any, key_map: Dict[int, int], all_config_keys: set[int]
) -> Any:
    """Recursively transform values, replacing config keys with synnax keys."""
    if isinstance(value, dict):
        return {
            k: transform_value(v, key_map, all_config_keys) for k, v in value.items()
        }
    elif isinstance(value, list):
        return [transform_value(item, key_map, all_config_keys) for item in value]
    elif isinstance(value, int):
        # If in key map, use the synnax key
        if value in key_map:
            return key_map[value]
        # If it looks like a channel key (> 1000000) but not in our configs, set to 0
        elif value > 1000000:
            return 0
        # Otherwise keep the value as-is
        else:
            return value
    else:
        return value


def transform_schematic(
    schematic_path: Path, mapping_path: Path, output_path: Path
) -> None:
    """Transform schematic by replacing config keys with Synnax keys."""
    print(f"\n{'=' * 70}")
    print("SCHEMATIC TRANSFORMER")
    print(f"{'=' * 70}\n")

    # Load channel mapping
    print(f"Loading channel mapping from: {mapping_path}")
    key_map = load_channel_mapping(mapping_path)
    print(f"✓ Loaded {len(key_map)} channel mappings\n")

    # Load schematic
    print(f"Loading schematic from: {schematic_path}")
    with open(schematic_path) as f:
        schematic = json.load(f)
    print(f"✓ Loaded schematic\n")

    # Transform schematic
    print("Transforming channel keys...")
    all_config_keys = set(key_map.keys())
    transformed = transform_value(schematic, key_map, all_config_keys)

    # Count and show replacements
    original_str = json.dumps(schematic)
    transformed_str = json.dumps(transformed)

    print(f"\nReplacement summary:")
    print(f"  Config keys in mapping: {len(key_map)}")

    # Count actual replacements
    replacements = 0
    for config_key in key_map.keys():
        if str(config_key) in original_str:
            replacements += 1

    print(f"  Config keys found in schematic: {replacements}")

    # Count keys set to 0 (not in mapping but > 1000000)
    zeros_set = transformed_str.count(": 0") - original_str.count(": 0")
    if zeros_set > 0:
        print(f"  Unknown channel keys replaced with 0: ~{zeros_set}")

    print(f"✓ Transformation complete\n")

    # Save transformed schematic
    print(f"Saving transformed schematic to: {output_path}")
    with open(output_path, "w") as f:
        json.dump(transformed, f, indent=2)
    print(f"✓ Saved successfully\n")

    print(f"{'=' * 70}")
    print("TRANSFORMATION COMPLETE")
    print(f"{'=' * 70}\n")


def main() -> None:
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="Transform schematic by replacing config channel keys with Synnax keys.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s schematic.json mapping.json
  %(prog)s ~/Desktop/fh_debug/CTS_Operator_CP.json ~/Desktop/fh_debug/channel_mapping.json
  %(prog)s schematic.json mapping.json --output transformed.json

This script reads a schematic JSON file and replaces all config channel keys
with their corresponding Synnax keys from the channel mapping file.

The script will:
  1. Load the channel mapping (config_key → synnax_key)
  2. Recursively scan the schematic for channel keys
  3. Replace config keys with Synnax keys
  4. Replace unknown keys (>1000000) with 0
  5. Save the transformed schematic with '_transformed' suffix
        """,
    )
    parser.add_argument(
        "schematic", type=str, help="Path to schematic JSON file to transform"
    )
    parser.add_argument("mapping", type=str, help="Path to channel_mapping.json file")
    parser.add_argument(
        "--output",
        type=str,
        help="Output path for transformed schematic (default: <schematic>_transformed.json)",
    )

    args = parser.parse_args()

    # Parse schematic path
    schematic_path = Path(args.schematic).expanduser().resolve()

    # Parse mapping path
    mapping_path = Path(args.mapping).expanduser().resolve()

    # Determine output path
    if args.output:
        output_path = Path(args.output).expanduser().resolve()
    else:
        # Add _transformed before the extension
        stem = schematic_path.stem
        suffix = schematic_path.suffix
        output_path = schematic_path.parent / f"{stem}_transformed{suffix}"

    # Validate inputs
    if not schematic_path.exists():
        print(f"✗ ERROR: Schematic not found: {schematic_path}")
        print(f"  Please ensure the schematic JSON exists.")
        sys.exit(1)

    if not mapping_path.exists():
        print(f"✗ ERROR: Channel mapping not found: {mapping_path}")
        print(f"  Please run sim_from_task_configs.py first to generate the mapping.")
        sys.exit(1)

    # Transform
    transform_schematic(schematic_path, mapping_path, output_path)
    print(f"Output: {output_path}")


if __name__ == "__main__":
    main()
