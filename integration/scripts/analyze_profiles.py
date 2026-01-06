#!/usr/bin/env python3
#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Analyze Console profiling data and generate comparison tables.

Usage:
    python scripts/analyze_profiles.py [--heap] [--format markdown|csv]

Options:
    --heap      Include heap snapshot analysis (slower, requires large file parsing)
    --format    Output format: markdown (default) or csv
"""

from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any


@dataclass
class HeapStats:
    """Statistics extracted from a V8 heap snapshot.

    :param node_count: Total number of heap nodes.
    :param edge_count: Total number of edges between nodes.
    :param total_self_size_mb: Total self size of all nodes in MB.
    :param total_retained_size_mb: Total retained size in MB.
    :param detached_nodes: Count of nodes with "Detached" in their name.
    :param type_counts: Count of nodes by type.
    :param error: Error message if parsing failed.
    """

    node_count: int = 0
    edge_count: int = 0
    total_self_size_mb: float = 0.0
    total_retained_size_mb: float = 0.0
    detached_nodes: int = 0
    type_counts: dict[str, int] | None = None
    error: str | None = None


class MetricsLoader:
    """Loads performance metrics from profile files.

    :param profiles_dir: Directory containing profile files.
    """

    def __init__(self, profiles_dir: Path) -> None:
        self._profiles_dir = profiles_dir

    def load_metrics(self) -> list[dict[str, Any]]:
        """Load all metrics.json files from the profiles directory.

        :returns: List of metrics dictionaries.
        """
        metrics_files = sorted(self._profiles_dir.glob("*.metrics.json"))
        results = []
        for f in metrics_files:
            with open(f) as fp:
                data = json.load(fp)
                data["_file"] = f.name
                results.append(data)
        return results

    def load_heap_snapshots(
        self, test_names: list[str]
    ) -> dict[str, HeapStats]:
        """Load and parse heap snapshots for given test names.

        :param test_names: List of test names to load snapshots for.
        :returns: Dictionary mapping test names to heap statistics.
        """
        heap_data: dict[str, HeapStats] = {}
        for test_name in test_names:
            snapshot_path = self._profiles_dir / f"{test_name}.heapsnapshot"
            if snapshot_path.exists():
                heap_data[test_name] = self._parse_heap_snapshot(snapshot_path)
                print(f"  Parsed {test_name}.heapsnapshot")
            else:
                print(f"  Skipped {test_name} (no heap snapshot)")
        return heap_data

    def _parse_heap_snapshot(self, snapshot_path: Path) -> HeapStats:
        """Parse a V8 heap snapshot for summary statistics.

        :param snapshot_path: Path to the heap snapshot file.
        :returns: HeapStats with parsed data or error.
        """
        try:
            with open(snapshot_path) as f:
                content = f.read()
                snapshot = json.loads(content)

            meta = snapshot.get("snapshot", {}).get("meta", {})
            node_count = snapshot.get("snapshot", {}).get("node_count", 0)
            edge_count = snapshot.get("snapshot", {}).get("edge_count", 0)

            node_fields = meta.get("node_fields", [])
            node_types = meta.get("node_types", [[]])[0]
            nodes = snapshot.get("nodes", [])
            strings = snapshot.get("strings", [])

            type_idx = node_fields.index("type") if "type" in node_fields else 0
            name_idx = node_fields.index("name") if "name" in node_fields else 1
            self_size_idx = (
                node_fields.index("self_size") if "self_size" in node_fields else 3
            )
            retained_size_idx = (
                node_fields.index("retained_size")
                if "retained_size" in node_fields
                else -1
            )

            field_count = len(node_fields)
            type_counts: dict[str, int] = {}
            total_self_size = 0
            total_retained_size = 0
            detached_count = 0

            for i in range(0, len(nodes), field_count):
                node_type_idx = nodes[i + type_idx]
                node_type = (
                    node_types[node_type_idx]
                    if node_type_idx < len(node_types)
                    else "unknown"
                )

                type_counts[node_type] = type_counts.get(node_type, 0) + 1
                total_self_size += nodes[i + self_size_idx]

                if retained_size_idx >= 0 and i + retained_size_idx < len(nodes):
                    total_retained_size += nodes[i + retained_size_idx]

                name_string_idx = nodes[i + name_idx]
                if name_string_idx < len(strings):
                    name = strings[name_string_idx]
                    if "Detached" in name:
                        detached_count += 1

            return HeapStats(
                node_count=node_count,
                edge_count=edge_count,
                total_self_size_mb=round(total_self_size / 1024 / 1024, 2),
                total_retained_size_mb=round(total_retained_size / 1024 / 1024, 2),
                detached_nodes=detached_count,
                type_counts=type_counts,
            )
        except Exception as e:
            return HeapStats(error=str(e))


class ReportFormatter:
    """Formats metrics data into various output formats."""

    def format_markdown_table(
        self,
        metrics: list[dict[str, Any]],
        heap_data: dict[str, HeapStats] | None = None,
    ) -> str:
        """Generate a markdown table from metrics data.

        :param metrics: List of metrics dictionaries.
        :param heap_data: Optional heap statistics keyed by test name.
        :returns: Markdown formatted table string.
        """
        headers = [
            "Test",
            "Heap Used (MB)",
            "Heap Total (MB)",
            "DOM Nodes",
            "Listeners",
            "Layout Ops",
            "Recalc Style",
            "Layout (ms)",
            "Style (ms)",
            "Script (ms)",
            "Task (ms)",
            "Detached Scripts",
            "Workers",
        ]

        if heap_data:
            headers.extend(["Heap Nodes", "Detached DOM"])

        rows = []
        for m in metrics:
            raw = m.get("raw", {})
            row = [
                m.get("test_name", "unknown"),
                str(int(m.get("memory", {}).get("js_heap_used_mb", 0))),
                str(int(m.get("memory", {}).get("js_heap_total_mb", 0))),
                str(m.get("dom", {}).get("nodes", 0)),
                str(m.get("dom", {}).get("js_event_listeners", 0)),
                str(m.get("layout", {}).get("layout_count", 0)),
                str(m.get("layout", {}).get("recalc_style_count", 0)),
                str(int(m.get("layout", {}).get("layout_duration_ms", 0))),
                str(int(m.get("layout", {}).get("recalc_style_duration_ms", 0))),
                str(int(m.get("script", {}).get("script_duration_ms", 0))),
                str(int(m.get("script", {}).get("task_duration_ms", 0))),
                str(int(raw.get("DetachedScriptStates", 0))),
                str(int(raw.get("WorkerGlobalScopes", 0))),
            ]

            if heap_data:
                test_name = m.get("test_name", "")
                hd = heap_data.get(test_name)
                row.append(str(hd.node_count if hd else "N/A"))
                row.append(str(hd.detached_nodes if hd else "N/A"))

            rows.append(row)

        col_widths = [len(h) for h in headers]
        for row in rows:
            for i, cell in enumerate(row):
                col_widths[i] = max(col_widths[i], len(cell))

        def format_row(cells: list[str]) -> str:
            return (
                "| "
                + " | ".join(c.ljust(col_widths[i]) for i, c in enumerate(cells))
                + " |"
            )

        separator = "|" + "|".join("-" * (w + 2) for w in col_widths) + "|"

        lines = [format_row(headers), separator]
        lines.extend(format_row(row) for row in rows)

        return "\n".join(lines)

    def format_csv(
        self,
        metrics: list[dict[str, Any]],
        heap_data: dict[str, HeapStats] | None = None,
    ) -> str:
        """Generate CSV output from metrics data.

        :param metrics: List of metrics dictionaries.
        :param heap_data: Optional heap statistics keyed by test name.
        :returns: CSV formatted string.
        """
        headers = [
            "test_name",
            "heap_used_mb",
            "heap_total_mb",
            "dom_nodes",
            "event_listeners",
            "layout_count",
            "recalc_style_count",
            "layout_duration_ms",
            "recalc_style_duration_ms",
            "script_duration_ms",
            "task_duration_ms",
            "detached_script_states",
            "worker_global_scopes",
            "array_buffer_contents",
            "resources",
            "v8_per_context_datas",
        ]

        if heap_data:
            headers.extend(["heap_node_count", "heap_edge_count", "detached_dom_nodes"])

        rows = [",".join(headers)]

        for m in metrics:
            raw = m.get("raw", {})
            row = [
                m.get("test_name", "unknown"),
                str(m.get("memory", {}).get("js_heap_used_mb", 0)),
                str(m.get("memory", {}).get("js_heap_total_mb", 0)),
                str(m.get("dom", {}).get("nodes", 0)),
                str(m.get("dom", {}).get("js_event_listeners", 0)),
                str(m.get("layout", {}).get("layout_count", 0)),
                str(m.get("layout", {}).get("recalc_style_count", 0)),
                str(m.get("layout", {}).get("layout_duration_ms", 0)),
                str(m.get("layout", {}).get("recalc_style_duration_ms", 0)),
                str(m.get("script", {}).get("script_duration_ms", 0)),
                str(m.get("script", {}).get("task_duration_ms", 0)),
                str(int(raw.get("DetachedScriptStates", 0))),
                str(int(raw.get("WorkerGlobalScopes", 0))),
                str(int(raw.get("ArrayBufferContents", 0))),
                str(int(raw.get("Resources", 0))),
                str(int(raw.get("V8PerContextDatas", 0))),
            ]

            if heap_data:
                test_name = m.get("test_name", "")
                hd = heap_data.get(test_name)
                row.append(str(hd.node_count if hd else ""))
                row.append(str(hd.edge_count if hd else ""))
                row.append(str(hd.detached_nodes if hd else ""))

            rows.append(",".join(row))

        return "\n".join(rows)

    def format_summary(self, metrics: list[dict[str, Any]]) -> str:
        """Generate summary statistics and potential concerns.

        :param metrics: List of metrics dictionaries.
        :returns: Markdown formatted summary string.
        """
        lines = ["\n## Summary Statistics\n"]

        heap_values = [m.get("memory", {}).get("js_heap_used_mb", 0) for m in metrics]
        dom_values = [m.get("dom", {}).get("nodes", 0) for m in metrics]
        layout_values = [m.get("layout", {}).get("layout_count", 0) for m in metrics]
        script_values = [
            m.get("script", {}).get("script_duration_ms", 0) for m in metrics
        ]

        lines.append(
            f"- **Heap Used**: {min(heap_values):.0f} - {max(heap_values):.0f} MB "
            f"(avg: {sum(heap_values)/len(heap_values):.0f} MB)"
        )
        lines.append(
            f"- **DOM Nodes**: {min(dom_values):,} - {max(dom_values):,} "
            f"(avg: {sum(dom_values)//len(dom_values):,})"
        )
        lines.append(
            f"- **Layout Ops**: {min(layout_values):,} - {max(layout_values):,}"
        )
        lines.append(
            f"- **Script Time**: {min(script_values):.0f} - {max(script_values):.0f} ms"
        )

        lines.append("\n## Potential Concerns\n")

        avg_dom = sum(dom_values) / len(dom_values)
        avg_layout = sum(layout_values) / len(layout_values)
        avg_script = sum(script_values) / len(script_values)

        concerns = []
        for m in metrics:
            test_name = m.get("test_name", "unknown")
            dom = m.get("dom", {}).get("nodes", 0)
            layout = m.get("layout", {}).get("layout_count", 0)
            script = m.get("script", {}).get("script_duration_ms", 0)
            raw = m.get("raw", {})
            workers = int(raw.get("WorkerGlobalScopes", 0))
            detached = int(raw.get("DetachedScriptStates", 0))

            if dom > avg_dom * 2:
                concerns.append(
                    f"- **{test_name}**: High DOM node count "
                    f"({dom:,} vs avg {avg_dom:,.0f})"
                )
            if layout > avg_layout * 2:
                concerns.append(
                    f"- **{test_name}**: High layout operations "
                    f"({layout:,} vs avg {avg_layout:,.0f})"
                )
            if script > avg_script * 2:
                concerns.append(
                    f"- **{test_name}**: High script duration "
                    f"({script:.0f}ms vs avg {avg_script:.0f}ms)"
                )
            if workers > 1:
                concerns.append(
                    f"- **{test_name}**: Multiple workers ({workers}) - check for cleanup"
                )
            if detached > 3:
                concerns.append(
                    f"- **{test_name}**: Elevated detached script states ({detached})"
                )

        if concerns:
            lines.append("\n".join(concerns))
        else:
            lines.append("No significant concerns detected.")

        return "\n".join(lines)


def main() -> None:
    """Main entry point for the profile analyzer."""
    parser = argparse.ArgumentParser(description="Analyze Console profiling data")
    parser.add_argument(
        "--heap", action="store_true", help="Include heap snapshot analysis"
    )
    parser.add_argument(
        "--format",
        choices=["markdown", "csv"],
        default="markdown",
        help="Output format",
    )
    parser.add_argument(
        "--profiles-dir",
        type=Path,
        default=Path(__file__).parent.parent / "profiles",
        help="Path to profiles directory",
    )
    args = parser.parse_args()

    profiles_dir = args.profiles_dir
    if not profiles_dir.exists():
        print(f"Error: Profiles directory not found: {profiles_dir}", file=sys.stderr)
        sys.exit(1)

    loader = MetricsLoader(profiles_dir)
    formatter = ReportFormatter()

    metrics = loader.load_metrics()
    if not metrics:
        print("Error: No metrics files found", file=sys.stderr)
        sys.exit(1)

    print(f"Found {len(metrics)} metrics files")

    heap_data: dict[str, HeapStats] | None = None
    if args.heap:
        print("Parsing heap snapshots (this may take a moment)...")
        test_names = [m.get("test_name", "") for m in metrics]
        heap_data = loader.load_heap_snapshots(test_names)

    if args.format == "csv":
        output_path = profiles_dir / "profile_summary.csv"
        content = formatter.format_csv(metrics, heap_data)
    else:
        output_path = profiles_dir / "profile_summary.md"
        content = "# Console Profile Summary\n\n"
        content += "## Performance Metrics Comparison\n\n"
        content += formatter.format_markdown_table(metrics, heap_data)
        content += formatter.format_summary(metrics)

    with open(output_path, "w") as f:
        f.write(content)

    print(f"Summary written to {output_path}")


if __name__ == "__main__":
    main()
