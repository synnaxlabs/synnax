#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that a graph Arc offers its node selectors in the Component toolbar."""

from console.case import ConsoleCase
from x import random_name

EXPECTED_GROUPS = ["Basic", "Telemetry", "Operators", "Flow control"]


class GraphToolbar(ConsoleCase):
    """Open a graph Arc and check its stage palette.

    The palette is the only way to place a node, and it reaches the screen through
    the Arc tab's registered toolbar. When that registration is missing the drawer
    silently falls back to its "no toolbar" content, so this drives the real path:
    open the Arc, open the drawer, and read the selectors back.
    """

    arc_name: str

    def setup(self) -> None:
        super().setup()
        self.arc_name = f"graph_toolbar_{random_name()}"
        self.client.arcs.create(name=self.arc_name, mode="graph")

    def teardown(self) -> None:
        self.console.arc.delete(self.arc_name)
        super().teardown()

    def run(self) -> None:
        arc = self.console.arc
        arc.open_graph(self.arc_name)
        arc.show_graph_toolbar()

        groups = arc.stage_group_names()
        for expected in EXPECTED_GROUPS:
            assert expected in groups, (
                f"stage group '{expected}' missing from the graph toolbar: {groups}"
            )

        basic = arc.stage_names()
        assert len(basic) > 0, "the Basic group offers no node selectors"
        assert "Constant" in basic, f"'Constant' missing from the Basic group: {basic}"

        # Selecting another group has to swap the selectors, not just the highlight.
        arc.select_stage_group("Telemetry")
        telemetry = arc.stage_names()
        assert len(telemetry) > 0, "the Telemetry group offers no node selectors"
        assert "Constant" not in telemetry, (
            f"the Basic group's selectors survived the switch: {telemetry}"
        )
