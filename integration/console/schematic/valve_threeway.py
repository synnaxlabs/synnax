#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from .valve import Valve


class ValveThreeWay(Valve):
    """Schematic three-way valve symbol"""

    def __init__(
        self,
        *,
        label: str,
        state_channel: str,
        command_channel: str,
        symbol_type: str = "Three Way",
        show_control_chip: bool = True,
        rotatable: bool = True,
    ):
        """Initialize a three-way valve symbol with configuration.

        Args:
            label: Display label for the symbol
            state_channel: Channel name for valve state
            command_channel: Channel name for valve commands
            show_control_chip: Whether to show the control chip (optional)
            symbol_type: The type of symbol (default: "Three Way")
            rotatable: Whether the symbol can be rotated (default: True)
        """
        super().__init__(
            label=label,
            state_channel=state_channel,
            command_channel=command_channel,
            show_control_chip=show_control_chip,
            symbol_type=symbol_type,
            rotatable=rotatable,
        )
