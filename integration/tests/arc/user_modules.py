#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from string import Template

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc import ArcCase

LIBRARY_SOURCE = """
export func seven() {t u8} i64 { return 7 }

export sequence seq {
    stage a {
        1 -> um_seq_state
    }
}

export stage stg {
    1 -> um_stage_state
}
"""

# The function reads a channel, so its call node fires as the channel updates.
REACTIVE_SOURCE = """
export func describe() str { return f"Data: {um_data}" }
"""

# A chain of three modules, one directory level deeper each. Each one imports the
# next by the path load_module gave it, binds the last segment, and adds one to
# its value.
CHAIN_A_SOURCE = Template("""
import $inner

export func value() {t u8} i64 { return $inner_name.value() + 1 }
""")

CHAIN_B_SOURCE = Template("""
import $inner

export func value() i64 { return $inner_name.value() + 1 }
""")

CHAIN_C_SOURCE = """
export func value() i64 { return 40 }
"""

# Each module is imported by the name load_module gives it.
AUTOMATION_SOURCE = Template("""
import ($library $reactive $chain)

authority 200

um_start -> $library.seven() -> um_fn_out
um_start == 1 => $library.seq
um_start == 1 => $library.stg
$reactive.describe() -> um_text_out
um_start -> $chain_name.value() -> um_chain_out
""")


def last_segment(path: str) -> str:
    """Return the name an import of path binds."""
    return path.rsplit(".", 1)[-1]


class UserModules(ArcCase):
    """An automation runs code it imports from user modules.

    1. A library module, a reactive module, and a chain of three modules are
       created. The chain sits at ``level1/a``, ``level1/level2/b``, and
       ``level1/level2/level3/c`` below Arcs.
    2. The automation calls a function, activates a sequence and a stage, routes a
       channel-reading function's value to a channel, and calls down the chain.
    3. The start trigger produces the function's value, both scope writes, and the
       chain's value.
    4. A write to ``um_data`` shows up formatted on ``um_text_out``.
    """

    arc_source = AUTOMATION_SOURCE.template
    arc_name_prefix = "ArcUserModules"
    start_cmd_channel = "um_start"
    subscribe_channels = [
        "um_fn_out",
        "um_seq_state",
        "um_stage_state",
        "um_text_out",
        "um_chain_out",
    ]

    def setup(self) -> None:
        create_virtual_channel(self.client, "um_fn_out", data_type=sy.DataType.INT64)
        create_virtual_channel(self.client, "um_seq_state", data_type=sy.DataType.UINT8)
        create_virtual_channel(
            self.client, "um_stage_state", data_type=sy.DataType.UINT8
        )
        create_virtual_channel(self.client, "um_data", data_type=sy.DataType.INT64)
        create_virtual_channel(self.client, "um_text_out", data_type=sy.DataType.STRING)
        create_virtual_channel(self.client, "um_chain_out", data_type=sy.DataType.INT64)
        super().setup()
        self.set_manual_timeout(60)

    def run(self) -> None:
        self._retrieve_rack()
        level1 = self.create_directory("level1")
        level2 = self.create_directory("level2", parent=level1)
        level3 = self.create_directory("level3", parent=level2)
        chain_c = self.load_module(
            CHAIN_C_SOURCE, name_prefix="um_chain_c", parent=level3
        )
        chain_b = self.load_module(
            CHAIN_B_SOURCE.substitute(inner=chain_c, inner_name=last_segment(chain_c)),
            name_prefix="um_chain_b",
            parent=level2,
        )
        chain_a = self.load_module(
            CHAIN_A_SOURCE.substitute(inner=chain_b, inner_name=last_segment(chain_b)),
            name_prefix="um_chain_a",
            parent=level1,
        )
        source = AUTOMATION_SOURCE.substitute(
            library=self.load_module(LIBRARY_SOURCE, name_prefix="um_library"),
            reactive=self.load_module(REACTIVE_SOURCE, name_prefix="um_reactive"),
            chain=chain_a,
            chain_name=last_segment(chain_a),
        )
        self.arc_name = self.load_arc(
            source, name_prefix=self.arc_name_prefix, trigger=self.start_cmd_channel
        )
        self.verify_sequence_execution()

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for the exported function's value")
        self.wait_for_eq("um_fn_out", 7, timeout=5 * sy.TimeSpan.SECOND)
        self.log("Waiting for the exported sequence's write")
        self.wait_for_eq("um_seq_state", 1, timeout=5 * sy.TimeSpan.SECOND)
        self.log("Waiting for the exported stage's write")
        self.wait_for_eq("um_stage_state", 1, timeout=5 * sy.TimeSpan.SECOND)
        self.log("Waiting for the chain's value")
        self.wait_for_eq("um_chain_out", 42, timeout=5 * sy.TimeSpan.SECOND)
        self.log("Writing um_data=42; the text function must format it")
        self.writer.write("um_data", 42)
        self.wait_for_eq("um_text_out", "Data: 42", timeout=5 * sy.TimeSpan.SECOND)
