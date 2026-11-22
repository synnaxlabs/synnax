#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import expect

from console.case import ConsoleCase
from synnax import enumeration, project, schematic
from x import random_name, spatial

# A built-in that no other test edits.
_BUILTIN = "Switch"
_BUILTIN_ENTRIES = [("off", "Off", 0.0), ("on", "On", 1.0)]


class EnumerationLifecycle(ConsoleCase):
    """Test creating, editing, restoring, and deleting enumerations in Settings."""

    name: str
    entries: list[str]
    replaced: str
    users: project.Project

    def setup(self) -> None:
        super().setup()
        self.name = random_name()
        self.entries = [random_name() for _ in range(3)]
        self.replaced = random_name()
        replaced = self.client.enumerations.create(
            enumeration.Enumeration(name=self.replaced)
        )
        # A select that uses the enumeration makes its delete ask for a replacement.
        self.users = self.client.projects.create(name=random_name())
        select = schematic.SelectNodeConfig(enumeration=replaced.key)
        self.client.schematics.create(
            schematic.Schematic(
                name=random_name(),
                project=self.users.key,
                nodes=[
                    schematic.Node(
                        key="select", position=spatial.XY(x=0, y=0), config=select
                    )
                ],
            )
        )

    def teardown(self) -> None:
        # The select in the project uses an enumeration until test_replace runs, and the
        # Core refuses to delete an enumeration in use.
        self.client.projects.delete(self.users.key)
        stale = self.client.enumerations.retrieve(names=[self.name, self.replaced])
        if stale:
            self.client.enumerations.delete([e.key for e in stale])
        # A run that fails after editing the built-in leaves it edited for the retry.
        enums = self.console.enumerations
        enums.search(_BUILTIN)
        if enums.menu_options(_BUILTIN, ["Restore default"]):
            enums.restore(_BUILTIN)
        super().teardown()

    def run(self) -> None:
        self.test_create()
        self.test_edit_entries()
        self.test_refuse_taken_value()
        self.test_add_and_remove_entry()
        self.test_rename()
        self.test_restore_builtin()
        self.test_replace()
        self.test_delete()

    def test_create(self) -> None:
        """Test that a new enumeration counts up the values of its entries."""
        self.log("Testing: Create enumeration")
        created = self.console.enumerations.create(self.name, entries=self.entries)
        assert [e.name for e in created.entries] == self.entries
        assert [e.value for e in created.entries] == [0, 1, 2]

    def test_edit_entries(self) -> None:
        """Test that each edit in place reaches the Core."""
        self.log("Testing: Edit entries in place")
        enums = self.console.enumerations
        first, second, third = self.entries
        renamed = random_name()
        enums.rename_entry(self.name, old_name=second, new_name=renamed)
        enums.set_value(self.name, third, 5)
        enums.toggle_abnormal(self.name, first)
        enums.set_color(self.name, first, "#FF0000")
        self.entries = [first, renamed, third]
        stored = enums.wait_for(
            self.name,
            lambda e: (
                [x.name for x in e.entries] == self.entries
                and e.entries[2].value == 5
                and e.entries[0].abnormal
                and e.entries[0].color is not None
            ),
            f"The Core did not store the edits to {self.name}",
        )
        color = stored.entries[0].color
        assert color is not None and color.type == "fixed"
        assert color.value == "#FF0000"

    def test_refuse_taken_value(self) -> None:
        """Test that a value another entry has stays in its cell, marked and unsent."""
        self.log("Testing: Refuse a taken value")
        enums = self.console.enumerations
        enums.set_value(self.name, self.entries[1], 5)
        cell = enums.section(self.name).get_by_label("Value 2", exact=True)
        expect(cell).to_have_value("5")
        expect(cell).to_have_attribute("aria-invalid", "true")
        stored = self.client.enumerations.retrieve(names=[self.name])[0]
        assert [e.value for e in stored.entries] == [0, 1, 5]

    def test_add_and_remove_entry(self) -> None:
        """Test the Add entry button and the Remove button of an entry."""
        self.log("Testing: Add and remove an entry")
        enums = self.console.enumerations
        added = random_name()
        enums.add_entry(self.name, added)
        stored = enums.wait_for(
            self.name,
            lambda e: [x.name for x in e.entries] == [*self.entries, added],
            f"The Core did not add {added}",
        )
        assert stored.entries[3].value == 6
        enums.remove_entry(self.name, added)
        enums.wait_for(
            self.name,
            lambda e: [x.name for x in e.entries] == self.entries,
            f"The Core did not remove {added}",
        )

    def test_rename(self) -> None:
        """Test renaming an enumeration from the menu of its heading."""
        self.log("Testing: Rename enumeration")
        new_name = random_name()
        self.console.enumerations.rename(old_name=self.name, new_name=new_name)
        self.name = new_name

    def test_restore_builtin(self) -> None:
        """Test that an edited built-in shows Edited and restores what Synnax ships."""
        self.log("Testing: Restore a built-in")
        enums = self.console.enumerations
        enums.search(_BUILTIN)
        heading = enums.heading(_BUILTIN)
        expect(heading.get_by_text("Built-in", exact=True)).to_be_visible()
        enums.rename_entry(_BUILTIN, old_name="Off", new_name=random_name())
        expect(heading.get_by_text("Edited", exact=True)).to_be_visible(timeout=5000)
        assert enums.menu_options(_BUILTIN, ["Restore default", "Delete"]) == [
            "Restore default"
        ]
        enums.restore(_BUILTIN)
        enums.wait_for(
            _BUILTIN,
            lambda e: [(x.key, x.name, x.value) for x in e.entries] == _BUILTIN_ENTRIES,
            f"The Core did not restore {_BUILTIN}",
        )
        expect(heading.get_by_text("Edited", exact=True)).to_have_count(0)

    def test_replace(self) -> None:
        """Test that deleting an enumeration in use moves its select to the pick."""
        self.log("Testing: Replace an enumeration")
        self.console.enumerations.search(self.replaced)
        # The built-in takes the select, so the enumeration that test_delete deletes
        # stays unused.
        self.console.enumerations.replace(self.replaced, with_name=_BUILTIN)

    def test_delete(self) -> None:
        """Test deleting an enumeration from the menu of its heading."""
        self.log("Testing: Delete enumeration")
        self.console.enumerations.search(self.name)
        self.console.enumerations.delete(self.name)
