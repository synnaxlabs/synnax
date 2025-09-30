#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
import synnax as sy
from typing import Literal
from console.console import Console

class Voltage():

    def __init__(self,
        console: Console,
        device: str,
        terminal_config: Literal[
            "Default",
            "Differential",
            "Pseudo-Differential",
            "Referenced Single Ended",
            "Non-Referenced Single Ended",
            ] = "Default",
        min_val: float = 0,
        max_val: float = 1,
        custom_scale: Literal[
            "None",
            "Linear",
            "Map",
            "Table",
            ] = "None"
        ) -> None:

        console.click_btn("Channel Type")
        console.select_from_dropdown("Voltage")

        console.click_btn("Device")
        console.select_from_dropdown(device)

        if console.check_for_modal():
            console.fill_input_field("Name","E103")
            console.META_ENTER
            console.fill_input_field("Identifier","E103Mod1")
            console.META_ENTER
        else:
            print('No modal,device is already configured')

        console.click_btn("Terminal Configuration")
        console.select_from_dropdown(terminal_config)

        console.fill_input_field("Minimum Value", str(min_val))
        console.fill_input_field("Maximum Value", str(max_val))

        console.click_btn("Custom Scaling")
        console.select_from_dropdown(custom_scale)
