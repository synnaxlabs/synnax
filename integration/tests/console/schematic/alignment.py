#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import math
from typing import Literal

import synnax as sy
from playwright.sync_api import FloatRect

from console.case import ConsoleCase
from console.schematic import (
    Schematic,
    Setpoint,
    Symbol,
    Valve,
    ValveThreeWay,
    ValveThreeWayBall,
)
from console.schematic.symbol import (
    box_bottom,
    box_center_x,
    box_center_y,
    box_left,
    box_right,
    box_top,
)

CHANNEL_NAME = "alignment_command_channel"

AlignmentType = Literal["vertical", "horizontal", "left", "right", "top", "bottom"]
DistributionType = Literal["horizontal", "vertical"]
RotationType = Literal["clockwise", "counterclockwise"]

ALIGNMENT_EXTRACTORS = {
    "left": box_left,
    "right": box_right,
    "top": box_top,
    "bottom": box_bottom,
    "horizontal": lambda pos: pos["x"] + pos["width"] / 2,
    "vertical": lambda pos: pos["y"] + pos["height"] / 2,
}


def assert_alignment(
    symbols: list[Symbol],
    alignment: AlignmentType,
    tolerance: float | None = None,
) -> None:
    """Assert that all symbols are aligned along the specified axis."""
    if len(symbols) < 2:
        raise ValueError("At least 2 symbols are required for alignment assertion")

    positions = [symbol.position for symbol in symbols]

    if tolerance is None:
        if alignment in ["horizontal", "vertical"]:
            tolerance = 15.0
        else:
            tolerance = 3.0

    extractor = ALIGNMENT_EXTRACTORS[alignment]
    first_coord = extractor(positions[0])

    for i, pos in enumerate(positions):
        coord = extractor(pos)
        diff = abs(coord - first_coord)
        assert diff <= tolerance, (
            f"Symbol {i} ('{symbols[i].label}') is not aligned on {alignment}!\n"
            f"Expected: {first_coord} (±{tolerance})\n"
            f"Actual: {coord}\n"
            f"Difference: {diff}\n"
            f"All symbols: {[s.label for s in symbols]}"
        )


def assert_distribution(
    symbols: list[Symbol],
    distribution: DistributionType,
    tolerance: float | None = None,
) -> None:
    """Assert that symbols are evenly distributed along the specified axis."""
    if len(symbols) < 3:
        raise ValueError("At least 3 symbols are required for distribution assertion")

    if tolerance is None:
        tolerance = 3.0 if distribution == "horizontal" else 20.0

    positions = [symbol.position for symbol in symbols]

    if distribution == "horizontal":
        sorted_data = sorted(zip(symbols, positions), key=lambda x: box_left(x[1]))
        sorted_symbols = [item[0] for item in sorted_data]
        sorted_positions = [item[1] for item in sorted_data]

        gaps = []
        for i in range(len(sorted_positions) - 1):
            current_right = box_right(sorted_positions[i])
            next_left = box_left(sorted_positions[i + 1])
            gaps.append(next_left - current_right)

        first_gap = gaps[0]
        for i, gap in enumerate(gaps):
            diff = abs(gap - first_gap)
            assert diff <= tolerance, (
                f"Horizontal gap {i} is not equal to first gap!\n"
                f"Expected gap: {first_gap} (±{tolerance})\n"
                f"Actual gap: {gap}\n"
                f"Difference: {diff}\n"
                f"Gap between '{sorted_symbols[i].label}' and '{sorted_symbols[i + 1].label}'"
            )
    else:
        sorted_data = sorted(zip(symbols, positions), key=lambda x: box_top(x[1]))
        sorted_symbols = [item[0] for item in sorted_data]
        sorted_positions = [item[1] for item in sorted_data]

        gaps = []
        for i in range(len(sorted_positions) - 1):
            current_bottom = box_bottom(sorted_positions[i])
            next_top = box_top(sorted_positions[i + 1])
            gaps.append(next_top - current_bottom)

        first_gap = gaps[0]
        for i, gap in enumerate(gaps):
            diff = abs(gap - first_gap)
            assert diff <= tolerance, (
                f"Vertical gap {i} is not equal to first gap!\n"
                f"Expected gap: {first_gap} (±{tolerance})\n"
                f"Actual gap: {gap}\n"
                f"Difference: {diff}\n"
                f"Gap between '{sorted_symbols[i].label}' and '{sorted_symbols[i + 1].label}'"
            )


def assert_rotation(
    symbols: list[Symbol],
    initial_positions: list[FloatRect],
    direction: RotationType,
    group: bool = False,
    tolerance: float = 3.0,
) -> None:
    """Assert that symbols have been rotated correctly.

    For individual rotation (group=False):
        - Verifies that symbol dimensions have changed (rotation affects visual appearance)
        - For symbols with rotation capability, the visual rotation should be apparent

    For group rotation (group=True):
        - Verifies that symbols have moved to new positions around the group center
        - Checks that the spatial arrangement has changed according to 90-degree rotation

    Args:
        symbols: List of symbols that were rotated
        initial_positions: List of Box objects before rotation (from symbol.position)
        direction: The rotation direction ('clockwise' or 'counterclockwise')
        group: If True, expects group rotation. If False, expects individual rotation.
        tolerance: Maximum allowed difference in pixels for position comparisons (default: 3.0)

    Raises:
        ValueError: If the number of symbols doesn't match initial_positions
        AssertionError: If rotation was not applied correctly
    """
    if len(symbols) != len(initial_positions):
        raise ValueError(
            f"Number of symbols ({len(symbols)}) must match "
            f"number of initial positions ({len(initial_positions)})"
        )

    current_positions = [symbol.position for symbol in symbols]

    _assert_individual_rotation_dimensions(
        symbols, initial_positions, current_positions, tolerance
    )

    if group:
        _assert_group_rotation_transform(
            symbols, initial_positions, current_positions, direction, tolerance
        )
    else:
        _assert_individual_rotation_ordering(
            symbols, initial_positions, current_positions
        )


def _assert_individual_rotation_dimensions(
    symbols: list[Symbol],
    initial_positions: list[FloatRect],
    current_positions: list[FloatRect],
    tolerance: float,
) -> None:
    """Assert that symbol dimensions changed correctly after individual rotation.

    For rotatable symbols: width and height should be swapped.
    For non-rotatable symbols: dimensions should remain unchanged.
    """
    for i, symbol in enumerate(symbols):
        initial_pos = initial_positions[i]
        current_pos = current_positions[i]

        if symbol.rotatable:
            # For rotatable symbols, dimensions should be swapped
            width_diff = abs(current_pos["width"] - initial_pos["height"])
            height_diff = abs(current_pos["height"] - initial_pos["width"])

            assert width_diff <= tolerance and height_diff <= tolerance, (
                f"Symbol {i} ({type(symbol).__name__}) dimensions not swapped after rotation!\n"
                f"Initial: width={initial_pos['width']:.1f}, height={initial_pos['height']:.1f}\n"
                f"Current: width={current_pos['width']:.1f}, height={current_pos['height']:.1f}\n"
                f"Expected: width={initial_pos['height']:.1f}, height={initial_pos['width']:.1f}\n"
                f"Difference: width_diff={width_diff:.1f}, height_diff={height_diff:.1f}\n"
                f"Tolerance: {tolerance}px"
            )
        else:
            # For non-rotatable symbols, dimensions should remain the same
            width_diff = abs(current_pos["width"] - initial_pos["width"])
            height_diff = abs(current_pos["height"] - initial_pos["height"])

            assert width_diff <= tolerance and height_diff <= tolerance, (
                f"Symbol {i} ({type(symbol).__name__}) dimensions changed after rotation attempt!\n"
                f"This symbol type cannot be rotated.\n"
                f"Initial: width={initial_pos['width']:.1f}, height={initial_pos['height']:.1f}\n"
                f"Current: width={current_pos['width']:.1f}, height={current_pos['height']:.1f}\n"
                f"Expected: width={initial_pos['width']:.1f}, height={initial_pos['height']:.1f}\n"
                f"Difference: width_diff={width_diff:.1f}, height_diff={height_diff:.1f}\n"
                f"Tolerance: {tolerance}px"
            )


def _assert_individual_rotation_ordering(
    symbols: list[Symbol],
    initial_positions: list[FloatRect],
    current_positions: list[FloatRect],
) -> None:
    """Assert that relative ordering is preserved after individual rotation.

    Only checks ordering for symbols that are clearly separated (>15px).
    Symbols close together are allowed to swap due to center position shifts during rotation.
    """
    ordering_separation_threshold = 15.0  # pixels

    def check_axis_ordering(alignment: AlignmentType) -> None:
        extractor = ALIGNMENT_EXTRACTORS[alignment]
        violations: list[dict[str, int | float]] = []
        for i in range(len(symbols)):
            for j in range(i + 1, len(symbols)):
                initial_i_val = extractor(initial_positions[i])
                initial_j_val = extractor(initial_positions[j])
                current_i_val = extractor(current_positions[i])
                current_j_val = extractor(current_positions[j])

                initial_diff = initial_j_val - initial_i_val
                current_diff = current_j_val - current_i_val

                if abs(initial_diff) > ordering_separation_threshold:
                    if (initial_diff > 0 and current_diff < 0) or (
                        initial_diff < 0 and current_diff > 0
                    ):
                        violations.append(
                            {
                                "i": i,
                                "j": j,
                                "initial_i": initial_i_val,
                                "initial_j": initial_j_val,
                                "current_i": current_i_val,
                                "current_j": current_j_val,
                            }
                        )

        if violations:
            msg = f"{alignment.capitalize()} ordering changed for well-separated symbols!\n"
            for v in violations:
                i_idx = int(v["i"])
                j_idx = int(v["j"])
                msg += f"  Symbols {i_idx} ({type(symbols[i_idx]).__name__}) and {j_idx} ({type(symbols[j_idx]).__name__}) swapped:\n"
                msg += f"    Initial: {v['initial_i']:.1f} vs {v['initial_j']:.1f} (diff: {abs(float(v['initial_j']) - float(v['initial_i'])):.1f}px)\n"
                msg += f"    Current: {v['current_i']:.1f} vs {v['current_j']:.1f} (diff: {abs(float(v['current_j']) - float(v['current_i'])):.1f}px)\n"
            raise AssertionError(msg)

    check_axis_ordering("horizontal")
    check_axis_ordering("vertical")


def _assert_group_rotation_transform(
    symbols: list[Symbol],
    initial_positions: list[FloatRect],
    current_positions: list[FloatRect],
    direction: RotationType,
    tolerance: float,
) -> None:
    """Assert that group rotation correctly transformed symbol positions.

    Uses angular displacement to verify rotation. Each symbol should have
    rotated by approximately ±90 degrees around the group center.
    """
    # Calculate initial center
    initial_center_x = sum(box_center_x(pos) for pos in initial_positions) / len(
        initial_positions
    )
    initial_center_y = sum(box_center_y(pos) for pos in initial_positions) / len(
        initial_positions
    )

    # Calculate current center
    current_center_x = sum(box_center_x(pos) for pos in current_positions) / len(
        current_positions
    )
    current_center_y = sum(box_center_y(pos) for pos in current_positions) / len(
        current_positions
    )

    expected_angle = 90.0 if direction == "clockwise" else -90.0
    angular_tolerance = 17.0

    for i in range(len(symbols)):

        init_x = box_center_x(initial_positions[i]) - initial_center_x
        init_y = box_center_y(initial_positions[i]) - initial_center_y

        curr_x = box_center_x(current_positions[i]) - current_center_x
        curr_y = box_center_y(current_positions[i]) - current_center_y

        initial_angle = math.degrees(math.atan2(init_y, init_x))
        current_angle = math.degrees(math.atan2(curr_y, curr_x))

        angular_displacement = current_angle - initial_angle
        if angular_displacement > 180:
            angular_displacement -= 360
        elif angular_displacement < -180:
            angular_displacement += 360

        angle_diff = abs(angular_displacement - expected_angle)

        assert angle_diff <= angular_tolerance, (
            f"Symbol {i} ({type(symbols[i]).__name__}) not rotated correctly in group!\n"
            f"Direction: {direction}\n"
            f"Initial angle: {initial_angle:.1f}°\n"
            f"Current angle: {current_angle:.1f}°\n"
            f"Angular displacement: {angular_displacement:.1f}°\n"
            f"Expected: {expected_angle:.1f}° ± {angular_tolerance:.1f}°\n"
            f"Difference: {angle_diff:.1f}°\n"
            f"Initial center: ({initial_center_x:.1f}, {initial_center_y:.1f})\n"
            f"Current center: ({current_center_x:.1f}, {current_center_y:.1f})"
        )


INDEX_NAME = "alignment_idx_channel"


class Alignment(ConsoleCase):
    """
    Test the alignment of symbols in the schematic
    """

    def setup(self) -> None:

        super().setup()

        index_ch = self.client.channels.create(
            name=INDEX_NAME,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        cmd_ch = self.client.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.FLOAT64,
            is_index=False,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        console = self.console
        client = self.client

        schematic = Schematic(console, "set_output_schematic")

        # Set up Symbols
        valve_threeway = schematic.create_symbol(
            ValveThreeWay(
                label=CHANNEL_NAME,
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve_threeway.move(delta_x=-150, delta_y=0)

        valve_threeway_ball = schematic.create_symbol(
            ValveThreeWayBall(
                label=CHANNEL_NAME,
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve_threeway_ball.move(delta_x=150, delta_y=-20)

        valve = schematic.create_symbol(
            Valve(
                label=CHANNEL_NAME,
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve.move(delta_x=0, delta_y=50)

        setpoint = schematic.create_symbol(
            Setpoint(label=CHANNEL_NAME, channel_name=CHANNEL_NAME)
        )
        setpoint.move(delta_x=-210, delta_y=0)

        symbols = [setpoint, valve_threeway, valve_threeway_ball, valve]

        self.log("Align Vertical")
        schematic.align(symbols, "vertical")
        assert_alignment(symbols, "vertical")

        self.log("Distribute Horizontal")
        schematic.distribute(symbols, "horizontal")
        assert_distribution(symbols, "horizontal")

        self.log("Align Horizontal")
        valve_threeway.move(delta_x=0, delta_y=-100)
        valve_threeway_ball.move(delta_x=0, delta_y=100)
        schematic.align(symbols, "horizontal")
        assert_alignment(symbols, "horizontal")

        self.log("Distribute Vertical")
        schematic.distribute(symbols, "vertical")
        assert_distribution(symbols, "vertical")

        self.log("Align Left")
        schematic.align(symbols, "left")
        assert_alignment(symbols, "left")

        self.log("Align Right")
        schematic.align(symbols, "right")
        assert_alignment(symbols, "right")

        self.log("Align Top")
        valve_threeway.move(delta_x=-150, delta_y=0)
        valve_threeway_ball.move(delta_x=150, delta_y=0)
        schematic.align(symbols, "top")
        assert_alignment(symbols, "top")
        schematic.distribute(symbols, "horizontal")
        assert_distribution(symbols, "horizontal")

        self.log("Align Bottom")
        valve.move(delta_x=0, delta_y=-20)
        valve_threeway.move(delta_x=0, delta_y=30)
        schematic.align(symbols, "bottom")
        assert_alignment(symbols, "bottom")

        self.log("Rotate Individual Clockwise")
        initial_positions = [symbol.position for symbol in symbols]
        schematic.rotate(symbols, "clockwise", group=False)
        assert_rotation(symbols, initial_positions, "clockwise", group=False)

        self.log("Rotate Individual Counterclockwise")
        initial_positions = [symbol.position for symbol in symbols]
        schematic.rotate(symbols, "counterclockwise", group=False)
        assert_rotation(symbols, initial_positions, "counterclockwise", group=False)

        self.log("Rotate Group Clockwise")
        initial_positions = [symbol.position for symbol in symbols]
        schematic.rotate(symbols, "clockwise", group=True)
        assert_rotation(symbols, initial_positions, "clockwise", group=True)

        self.log("Rotate Group Counterclockwise")
        initial_positions = [symbol.position for symbol in symbols]
        schematic.rotate(symbols, "counterclockwise", group=True)
        assert_rotation(symbols, initial_positions, "counterclockwise", group=True)

        schematic.screenshot()
