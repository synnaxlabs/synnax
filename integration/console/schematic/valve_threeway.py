#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.schematic.valve import Valve


class ValveThreeWay(Valve):
    """Schematic three-way valve symbol"""

    def __init__(
        self,
        *,
        label: str,
        state_channel: str,
        command_channel: str,
        show_control_chip: bool = True,
    ):
        super().__init__(
            label=label,
            state_channel=state_channel,
            command_channel=command_channel,
            show_control_chip=show_control_chip,
            symbol_type="Three way",
            rotatable=True,
        )
