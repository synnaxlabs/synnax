#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

import synnax as sy
from console.case import ConsoleCase
from x import random_name


class ArcRename(ConsoleCase):
    """Validate arc tab name synchronization in both directions."""

    _arc_keys: list[UUID]

    def setup(self) -> None:
        super().setup()
        self._arc_keys = []

    def teardown(self) -> None:
        if self._arc_keys:
            try:
                self.client.arcs.delete(self._arc_keys)
            except Exception:
                pass
        super().teardown()

    def run(self) -> None:
        self.test_tab_to_server()
        self.test_server_to_tab()

    def _create_arc_via_ui(self) -> tuple[str, UUID]:
        """Create an arc via the UI and return (name, key)."""
        name = f"arc_{random_name()}"
        self.console.arc.create(name=name, source="", mode="Text")
        arc = self.client.arcs.retrieve(name=name)
        self._arc_keys.append(arc.key)
        return name, arc.key

    def test_tab_to_server(self) -> None:
        """Renaming the arc tab should persist via Arc.useRename to the server."""
        self.log("arc tab → server")
        original, key = self._create_arc_via_ui()
        renamed = f"{original}_tab_renamed"

        self.console.layout.rename_tab(old_name=original, new_name=renamed)

        retrieved = self.client.arcs.retrieve(key=key)
        assert retrieved.name == renamed, (
            f"server arc should be renamed to '{renamed}', got '{retrieved.name}'"
        )

    def test_server_to_tab(self) -> None:
        """A server-side rename should propagate to the open arc tab."""
        self.log("arc server → tab")
        original, key = self._create_arc_via_ui()
        renamed = f"{original}_server_renamed"

        # Arcs are renamed via re-create with the same key; this fires the
        # arc set channel which the renderer's useRetrieveObservableName
        # subscription should pick up and dispatch as a Layout.rename.
        self.client.arcs.create(sy.Arc(key=key, name=renamed))

        self.console.layout.get_tab(renamed).wait_for(state="visible", timeout=5000)
