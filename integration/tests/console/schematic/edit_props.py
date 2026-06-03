#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from console.case import ConsoleCase
from console.schematic import Button, Symbol, Value
from console.schematic.schematic import PropertyDict, Schematic

CHANNEL_NAME = "button_cmd"
INDEX_NAME = "button_idx"

# Display names of every schematic edge variant, mirroring the variant selector
# dropdown (pluto SELECT_DATA in edge/common/segmented/Form.tsx). A name whose key
# does not resolve in the edge registry crashes the schematic graph view, so this
# list guards every option the user can pick.
EDGE_VARIANTS = [
    "Pipe",
    "Electrical",
    "Secondary",
    "Jacketed",
    "Hydraulic",
    "Pneumatic",
    "Data",
]


def assert_properties(
    schematic: Schematic, control_authority: int = 1, show_control_legend: bool = True
) -> None:
    """Assert the schematic properties match expected values."""
    props = schematic.get_properties()
    assert props["control_authority"] == control_authority, (
        f"Control authority mismatch! Actual: {props['control_authority']}, Expected: {control_authority}"
    )
    assert props["show_control_legend"] == show_control_legend, (
        f"Show control state legend mismatch! Actual: {props['show_control_legend']}, Expected: {show_control_legend}"
    )
    ...


def assert_symbol_properties(symbol: Symbol, expected_props: PropertyDict) -> None:
    """Assert the symbol properties match expected values."""
    actual_props = symbol.get_properties()
    assert actual_props == expected_props, (
        f"Props mismatch!\nActual: {actual_props}\nExpected: {expected_props}"
    )


class EditProps(ConsoleCase):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:
        client = self.client

        index_ch = client.channels.create(
            name=INDEX_NAME,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.UINT8,
            is_index=False,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

        schematic = self.console.workspace.create_schematic("edit_symbol_props")
        self._cleanup_pages.append(schematic.page_name)
        self.test_schematic_props(schematic)
        self.test_value_props(schematic)
        self.test_button_props(schematic)
        self.test_edge_variants(schematic)
        self.log("Test Complete")

    def test_schematic_props(self, schematic: Schematic) -> None:
        self.log("Test 0: Schematic Properties")

        self.log("0.1 Change Properties")
        assert_properties(schematic)

        schematic.set_properties(control_authority=7)
        assert_properties(schematic, control_authority=7, show_control_legend=True)

        schematic.set_properties(show_control_legend=False)
        assert_properties(schematic, control_authority=7, show_control_legend=False)

        schematic.set_properties(control_authority=128, show_control_legend=True)
        assert_properties(schematic, control_authority=128, show_control_legend=True)

        self.log("0.2 Acquire Control")
        button = schematic.create_symbol(
            Button(label=CHANNEL_NAME, channel_name=CHANNEL_NAME)
        )
        schematic.acquire_control()
        assert schematic.get_control_status() is True, (
            "Control status mismatch! Expected: True"
        )
        assert schematic.control_legend_visible is True, (
            "Control legend should be visible"
        )

        self.log("0.3 Hide Legend")
        schematic.release_control()
        assert schematic.get_control_status() is False, (
            "Control status mismatch! Expected: False"
        )
        schematic.enable_edit()
        assert schematic.get_edit_status() is True, (
            "Edit status mismatch! Expected: True"
        )
        schematic.set_properties(show_control_legend=False)
        schematic.acquire_control()
        assert schematic.control_legend_visible is False, (
            "Control legend should not be visible"
        )

        # Clean up schematic
        schematic.release_control()
        schematic.enable_edit()
        button.delete()

    def test_value_props(self, schematic: Schematic) -> None:
        self.log("Test 1: Value Properties")

        self.log("1.1 Default")
        value = schematic.create_symbol(
            Value(label=f"{self.name}_uptime", channel_name=f"{self.name}_uptime")
        )
        default_props: PropertyDict = {
            "channel": f"{self.name}_uptime",
            "notation": "standard",
            "precision": 2,
            "averaging_window": 1,
            "stale_color": "#C29D0A",  # pluto-warning-m1
            "stale_timeout": 5,
        }
        assert_symbol_properties(value, default_props)

        self.log("1.2 Edited")
        expected_edited_props: PropertyDict = {
            "channel": f"{self.name}_time",
            "notation": "scientific",
            "precision": 4,
            "averaging_window": 4,
            "stale_color": "#FF0000",
            "stale_timeout": 10,
        }
        value.set_properties(
            channel_name=f"{self.name}_time",
            notation="scientific",
            precision=4,
            averaging_window=4,
            stale_color="#FF0000",
            stale_timeout=10,
        )
        assert_symbol_properties(value, expected_edited_props)
        value.delete()

        self.log("1.3 Non-Default")
        non_default_props: PropertyDict = {
            "channel": f"{self.name}_state",
            "notation": "engineering",
            "precision": 7,
            "averaging_window": 3,
            "stale_color": "#00FF00",
            "stale_timeout": 15,
        }
        non_default_value = schematic.create_symbol(
            Value(
                label=f"{self.name}_state",
                channel_name=f"{self.name}_state",
                notation="engineering",
                precision=7,
                averaging_window=3,
                stale_color="#00FF00",
                stale_timeout=15,
            )
        )
        assert_symbol_properties(non_default_value, non_default_props)
        non_default_value.delete()

    def test_button_props(self, schematic: Schematic) -> None:
        self.log("Test 2: Button Properties")

        self.log("2.1 Default")
        button = schematic.create_symbol(
            Button(label=CHANNEL_NAME, channel_name=CHANNEL_NAME)
        )

        expected_default_props: PropertyDict = {
            "channel": CHANNEL_NAME,
            "activation_delay": 0,
            "show_control_chip": True,
            "mode": "Fire",
        }
        assert_symbol_properties(button, expected_default_props)

        self.log("2.2 Edited")
        button.set_properties(
            channel_name=CHANNEL_NAME,
            activation_delay=4.2,
            show_control_chip=False,
            mode="Momentary",
        )
        expected_edited_props: PropertyDict = {
            "channel": CHANNEL_NAME,
            "activation_delay": 4.2,
            "show_control_chip": False,
            "mode": "Momentary",
        }
        assert_symbol_properties(button, expected_edited_props)
        button.delete()

        self.log("2.3 Non-Default")
        non_default_props: PropertyDict = {
            "channel": CHANNEL_NAME,
            "activation_delay": 2.3,
            "show_control_chip": True,
            "mode": "Pulse",
        }
        non_default_button = schematic.create_symbol(
            Button(
                label=CHANNEL_NAME,
                channel_name=CHANNEL_NAME,
                activation_delay=2.3,
                show_control_chip=True,
                mode="Pulse",
            )
        )
        assert_symbol_properties(non_default_button, non_default_props)
        non_default_button.delete()

    def test_edge_variants(self, schematic: Schematic) -> None:
        self.log("Test 3: Edge Variants")
        schematic.enable_edit()

        source = schematic.create_symbol(
            Value(label=f"{CHANNEL_NAME}_edge_a", channel_name=CHANNEL_NAME)
        )
        source.move(delta_x=-200, delta_y=0)
        target = schematic.create_symbol(
            Value(label=f"{CHANNEL_NAME}_edge_b", channel_name=CHANNEL_NAME)
        )
        target.move(delta_x=200, delta_y=0)

        schematic.connect_symbols(source, "right", target, "left")
        assert schematic.get_edge_count() == 1, (
            f"Expected one edge after connecting, got {schematic.get_edge_count()}"
        )

        self.log("3.1 Default")
        schematic.select_edge(source, "right", target, "left")
        default_variant = schematic.get_edge_variant()
        assert default_variant == "Pipe", (
            f"Default edge variant mismatch! Actual: {default_variant}, Expected: Pipe"
        )

        for variant in EDGE_VARIANTS:
            self.log(f"3.2 Switch to {variant}")
            schematic.select_edge(source, "right", target, "left")
            schematic.set_edge_variant(variant)

            # A variant whose key is missing from the registry throws during render
            # and the error boundary tears down the graph view, dropping the node
            # and edge elements. Their continued presence proves no crash occurred.
            assert schematic.get_edge_count() == 1, (
                f"Schematic crashed after switching edge to {variant}: "
                "edge no longer rendered"
            )
            assert schematic.get_symbol_count() == 2, (
                f"Schematic crashed after switching edge to {variant}: "
                "symbols no longer rendered"
            )

            applied = schematic.get_edge_variant()
            assert applied == variant, (
                f"Edge variant did not apply! Actual: {applied}, Expected: {variant}"
            )

        source.delete()
        target.delete()
