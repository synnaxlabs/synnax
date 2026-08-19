#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Import the Core's canonical golden files through the real Console flow.

The goldens under ``core/pkg/service/<type>/versions/testdata/`` are the byte-exact
inputs the Core's own migration tests decode: current-version envelopes, legacy
versioned envelopes, typeless Console states, and rejection cases. Importing them
through the palette exercises the full chain the Core tests cannot: file chooser,
server-side type resolution and migration, project parenting, tab opening, and
error surfacing.
"""

import json
import os
import shutil
import tempfile
import zipfile
from collections.abc import Iterator
from contextlib import contextmanager
from dataclasses import dataclass

from console.case import ConsoleCase
from console.log import Log
from console.page import ConsolePage
from console.plot import Plot
from console.schematic import Schematic
from console.schematic.symbol_toolbar import SymbolToolbar
from console.table import Table
from framework.run_dir import resolve_results_path
from framework.utils import (
    assert_envelope,
    get_core_fixture_path,
    named_envelope_copy,
)
from x import random_name


@dataclass
class Golden:
    """One importable golden file and the page class it must open as."""

    label: str
    relative_path: str
    page_class: type[ConsolePage]


GOLDENS = [
    Golden("lineplot_v5", "lineplot/versions/testdata/import_v5.json", Plot),
    Golden("lineplot_state", "lineplot/versions/testdata/import_v0_state.json", Plot),
    Golden("log_v2", "log/versions/testdata/import_v2.json", Log),
    Golden("log_v1", "log/versions/testdata/import_v1.json", Log),
    Golden("log_v0", "log/versions/testdata/import_v0.json", Log),
    Golden("log_state", "log/versions/testdata/import_v0_state.json", Log),
    Golden("table_v1", "table/versions/testdata/import_v1.json", Table),
    Golden("table_state", "table/versions/testdata/import_v0_state.json", Table),
    Golden("table_typed", "table/versions/testdata/import_typed_console.json", Table),
    Golden("schematic_v7", "schematic/versions/testdata/import_v7.json", Schematic),
    Golden(
        "schematic_state",
        "schematic/versions/testdata/import_v3_state.json",
        Schematic,
    ),
    Golden(
        "schematic_typed",
        "schematic/versions/testdata/import_typed_console.json",
        Schematic,
    ),
]

# One current-version golden per type, re-exported and re-imported to prove the
# Console round-trips what it ingests.
ROUND_TRIPS = {
    "lineplot_v5": "lineplot",
    "log_v2": "log",
    "table_v1": "table",
    "schematic_v7": "schematic",
}

# Arcs import through the same palette flow but surface in the Arc panel
# instead of the project tree.
ARC_GOLDENS = [
    ("arc_v3", "arc/versions/testdata/import_v3.json"),
    ("arc_typed", "arc/versions/testdata/import_typed_console.json"),
    ("arc_stamped", "arc/versions/testdata/import_typed_console_stamped.json"),
    (
        "arc_edgeless",
        "arc/versions/testdata/import_typed_console_stamped_edgeless.json",
    ),
    ("arc_v0_state", "arc/versions/testdata/import_v0_state.json"),
    ("arc_v2_state", "arc/versions/testdata/import_v2_state.json"),
]

# Symbols import through the symbols toolbar into a selected group, not the
# palette. Their envelope names surface unmodified in the fresh per-test group.
SYMBOL_GOLDENS = [
    ("schematic/symbol/versions/testdata/import_v2.json", "Server Symbol"),
    ("schematic/symbol/versions/testdata/import_console.json", "Console Symbol"),
]

# Frozen copies of the task file shape released Consoles exported, one per
# released shape of every task type. Tasks import as rackless drafts, so no
# driver is needed. Names fall back to the filename: the files carry no name
# header. The legacy arc task fixtures (arc.json, arc_py.json) are absent:
# their "arc" type header routes to the Arc automation importer in the imex
# registry, which rejects them ("not an Arc").
TASK_GOLDENS = [
    "ethercat_read",
    "ethercat_write",
    "http_read",
    "http_read_list",
    "http_write",
    "labjack_read",
    "labjack_read_py_no_scale",
    "labjack_write",
    "labjack_write_cmd_key",
    "modbus_read",
    "modbus_write",
    "ni_analog_read",
    "ni_analog_read_config_device",
    "ni_analog_read_py",
    "ni_analog_write",
    "ni_counter_read",
    "ni_counter_read_py",
    "ni_digital_read",
    "ni_digital_write",
    "opc_read",
    "opc_read_array",
    "opc_write",
    "pagerduty_alert",
]

VERSION_REJECTION = "newer than this Core supports"

REJECTIONS = [
    ("lineplot/versions/testdata/import_bad_version.json", VERSION_REJECTION),
    ("log/versions/testdata/import_bad_version.json", VERSION_REJECTION),
    ("table/versions/testdata/import_bad_version.json", VERSION_REJECTION),
    ("schematic/versions/testdata/import_bad_version.json", VERSION_REJECTION),
    ("arc/versions/testdata/import_bad_version.json", VERSION_REJECTION),
    ("log/versions/testdata/import_unrecognized.json", "not a log"),
    ("arc/versions/testdata/import_unrecognized.json", "not an Arc"),
    ("arc/versions/testdata/import_unrecognized_versionless.json", "not an Arc"),
]

# Frozen symbol group bundles: the legacy directory shape released Consoles
# exported (version 1 manifest listing its members) and the current server
# export (version 2 manifest beside member envelopes). Both hold the same two
# members.
GROUP_GOLDENS = [
    ("group_v1", "schematic/symbol/versions/testdata/import_group_v1"),
    ("group_v2", "schematic/symbol/versions/testdata/import_group_v2"),
]
GROUP_MEMBERS = ["Inlet", "Outlet"]

SYMBOL_REJECTIONS = [
    ("schematic/symbol/versions/testdata/import_bad_version.json", VERSION_REJECTION),
    ("schematic/symbol/versions/testdata/import_unrecognized.json", "not a symbol"),
    ("schematic/symbol/versions/testdata/import_dataless.json", "not a symbol"),
]


class ImportGoldens(ConsoleCase):
    """Import every Core golden through the real Console flow and verify."""

    _symbol_group: str | None = None

    def run(self) -> None:
        self.suffix = random_name()
        self._symbol_groups: list[str] = []
        for golden in GOLDENS:
            self.test_import_golden(golden)
        for label, relative_path in ARC_GOLDENS:
            self.test_import_arc_golden(label, relative_path)
        self.test_import_legacy_tasks()
        for relative_path, error_text in REJECTIONS:
            self.test_rejection(relative_path, error_text)
        self.test_import_symbols()
        self.test_import_symbol_groups()

    def teardown(self) -> None:
        groups: list[str] = getattr(self, "_symbol_groups", [])
        leftover = [g for g in (self._symbol_group, *groups) if g]
        for name in leftover:
            with self._try_to(f"delete symbol group {name}"):
                toolbar = SymbolToolbar(self.console.layout)
                toolbar.show()
                if toolbar.group_exists(name):
                    toolbar.delete_group(name)
        super().teardown()

    def test_import_golden(self, golden: Golden) -> None:
        """Import one golden, verify the page opens, and round-trip it."""
        self.log(f"Importing golden: {golden.label}")
        name = f"{golden.label}_{self.suffix}"
        path = get_core_fixture_path(golden.relative_path)
        self._import_as(path, name)
        self._cleanup_pages.append(name)

        page = golden.page_class.from_open_page(self.console.layout, self.client, name)
        assert page.is_pane_visible, f"{golden.label}: imported pane not visible"

        envelope_type = ROUND_TRIPS.get(golden.label)
        if envelope_type is not None:
            self.test_round_trip(golden, name, envelope_type)
        page.close()

    def test_round_trip(self, golden: Golden, name: str, envelope_type: str) -> None:
        """Export the imported page and import the export under a new name."""
        self.log(f"Round-tripping golden: {golden.label}")
        exported = self.console.pages.export(name)
        assert_envelope(exported, envelope_type=envelope_type, min_version=1, name=name)

        reimport_name = f"{golden.label}_rt_{self.suffix}"
        export_path = resolve_results_path(f"{name}_export.json")
        self._import_as(export_path, reimport_name)
        self._cleanup_pages.append(reimport_name)
        page = golden.page_class.from_open_page(
            self.console.layout, self.client, reimport_name
        )
        assert page.is_pane_visible, f"{golden.label}: round-tripped pane not visible"
        page.close()

    def _import_as(self, path: str, name: str) -> None:
        """Import ``path`` so the resulting page is named ``name``."""
        with named_envelope_copy(path, name) as tmp_path:
            self.console.pages.import_file(tmp_path, name)

    def test_import_arc_golden(self, label: str, relative_path: str) -> None:
        """Import one arc golden and verify it appears in the Arc panel."""
        self.log(f"Importing arc golden: {label}")
        name = f"{label}_{self.suffix}"
        path = get_core_fixture_path(relative_path)
        with named_envelope_copy(path, name) as tmp_path:
            self.console.layout.choose_import_file(tmp_path)
        self.console.arc.wait_for_item(name)
        tab = self.console.layout.get_tab(name)
        if tab.is_visible():
            self.console.layout.close_tab(name)
        self.console.arc.delete(name)

    def test_import_legacy_tasks(self) -> None:
        """Batch-import the frozen legacy task corpus through one file chooser."""
        self.log("Importing legacy task goldens")
        with tempfile.TemporaryDirectory() as tmp_dir:
            paths: list[str] = []
            names: list[str] = []
            for label in TASK_GOLDENS:
                src = get_core_fixture_path(f"task/testdata/legacy/{label}.json")
                name = f"{label}_{self.suffix}"
                path = os.path.join(tmp_dir, f"{name}.json")
                shutil.copyfile(src, path)
                paths.append(path)
                names.append(name)
            self.console.layout.choose_import_file(paths)
            for name in names:
                self.console.tasks.wait_for(name)
        # Every import opens a resource tab; drop them all before cleanup.
        self.console.close_all_tabs()
        self.console.tasks.delete_many(names)

    def test_import_symbols(self) -> None:
        """Import symbol goldens into a fresh group via the symbols toolbar."""
        self.log("Importing symbol goldens")
        schematic = self.console.pages.create(
            Schematic, f"symbol_goldens_{self.suffix}"
        )
        self._cleanup_pages.append(schematic.page_name)
        self._symbol_group = f"Golden Group {self.suffix}"
        toolbar = SymbolToolbar(self.console.layout)
        toolbar.show()
        self.console.notifications.close_all()
        toolbar.create_group(self._symbol_group)
        toolbar.select_group(self._symbol_group)

        for relative_path, symbol_name in SYMBOL_GOLDENS:
            self.log(f"Importing symbol golden: {relative_path}")
            toolbar.import_symbol(get_core_fixture_path(relative_path))
            assert toolbar.symbol_exists(symbol_name), (
                f"Symbol {symbol_name!r} should appear in the group after import"
            )

        for relative_path, error_text in SYMBOL_REJECTIONS:
            self.log(f"Importing symbol rejection golden: {relative_path}")
            toolbar.import_symbol(get_core_fixture_path(relative_path))
            if not self.console.notifications.wait_for("Failed to import symbol"):
                raise AssertionError(
                    f"Import of {os.path.basename(relative_path)} did not "
                    "surface a failure notification"
                )
            if not self.console.notifications.wait_for(error_text):
                raise AssertionError(
                    f"Import failure notification does not mention {error_text!r}"
                )
            self.console.notifications.close_all()

        toolbar.delete_group(self._symbol_group)
        self._symbol_group = None
        schematic.close()

    @contextmanager
    def _group_bundle(self, directory: str, name: str) -> Iterator[str]:
        """Zip of a bundle directory with its manifest renamed to ``name``.

        A bundle's manifest name becomes the imported group's name, and the
        Core is long-lived across runs, so every import takes a unique name.
        """
        with tempfile.TemporaryDirectory() as tmp_dir:
            zip_path = os.path.join(tmp_dir, f"{name}.zip")
            with zipfile.ZipFile(zip_path, "w") as zf:
                for file_name in sorted(os.listdir(directory)):
                    path = os.path.join(directory, file_name)
                    if file_name == "manifest.json":
                        with open(path, "r", encoding="utf-8") as f:
                            manifest = json.load(f)
                        manifest["name"] = name
                        zf.writestr(file_name, json.dumps(manifest))
                    else:
                        zf.write(path, file_name)
            yield zip_path

    def _import_group_as(
        self, toolbar: SymbolToolbar, directory: str, name: str
    ) -> None:
        """Import a bundle directory as a group named ``name`` and verify it."""
        with self._group_bundle(directory, name) as zip_path:
            toolbar.import_group(zip_path)
        if not self.console.notifications.wait_for(f'Imported symbol group "{name}"'):
            raise AssertionError(
                f"Import of group {name!r} did not surface a success notification"
            )
        self.console.notifications.close_all()
        self._symbol_groups.append(name)
        for member in GROUP_MEMBERS:
            assert toolbar.symbol_exists(member, select_group=name), (
                f"Symbol {member!r} should appear in imported group {name!r}"
            )

    def test_import_symbol_groups(self) -> None:
        """Import the group bundle goldens and round-trip the current format."""
        self.log("Importing symbol group goldens")
        schematic = self.console.pages.create(Schematic, f"group_goldens_{self.suffix}")
        self._cleanup_pages.append(schematic.page_name)
        toolbar = SymbolToolbar(self.console.layout)
        toolbar.show()
        self.console.notifications.close_all()

        for label, relative_path in GROUP_GOLDENS:
            self.log(f"Importing group golden: {label}")
            self._import_group_as(
                toolbar, get_core_fixture_path(relative_path), f"{label}_{self.suffix}"
            )

        self.log("Round-tripping symbol group")
        zip_path = toolbar.export_group(f"group_v2_{self.suffix}")
        self.console.notifications.close_all()
        with tempfile.TemporaryDirectory() as tmp_dir:
            with zipfile.ZipFile(zip_path) as zf:
                zf.extractall(tmp_dir)
            self._import_group_as(toolbar, tmp_dir, f"group_rt_{self.suffix}")

        for name in list(self._symbol_groups):
            toolbar.delete_group(name)
            self._symbol_groups.remove(name)
        schematic.close()

    def test_rejection(self, relative_path: str, error_text: str) -> None:
        """Import a golden the Core must reject and assert the error surfaces."""
        self.log(f"Importing rejection golden: {relative_path}")
        path = get_core_fixture_path(relative_path)
        self.console.pages.import_file_expect_error(path, error_text)
