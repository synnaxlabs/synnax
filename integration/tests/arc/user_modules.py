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
# its value. The first one's stage reads the second one's channel-backed variable.
CHAIN_A_SOURCE = Template("""
import $inner

export func value() {t u8} i64 { return $inner_name.value() + 1 }

export stage relay {
    $inner_name.doubled + 1 -> um_var_out
}
""")

CHAIN_B_SOURCE = Template("""
import $inner

export doubled := um_data * 2

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
um_start == 1 => $chain_name.relay
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
    4. A write to ``um_data`` shows up formatted on ``um_text_out``, and doubled plus
       one on ``um_var_out`` through the chain's stage.
    5. An edit to the library module drifts the automation, which names the library.
       A redeploy runs the edited function.
    6. Renaming the export at the bottom of the chain fails the compile of every
       module above it and drifts the automation. After a restore, the automation
       names the bottom module, and a redeploy runs the new value.
    7. An edit to the automation's own text names the automation.
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
        "um_var_out",
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
        create_virtual_channel(self.client, "um_var_out", data_type=sy.DataType.INT64)
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
        library = self.load_module(LIBRARY_SOURCE, name_prefix="um_library")
        source = AUTOMATION_SOURCE.substitute(
            library=library,
            reactive=self.load_module(REACTIVE_SOURCE, name_prefix="um_reactive"),
            chain=chain_a,
            chain_name=last_segment(chain_a),
        )
        self.arc_name = self.load_arc(
            source, name_prefix=self.arc_name_prefix, trigger=self.start_cmd_channel
        )
        self.verify_sequence_execution()
        self._verify_module_edit(library)
        self._verify_chain_edit(chain_a, chain_b, chain_c)
        self._verify_automation_edit(source)

    def _verify_module_edit(self, library: str) -> None:
        self.log("Editing the library module so seven() returns 8")
        self._edit_and_redeploy(
            library,
            LIBRARY_SOURCE.replace("return 7", "return 8"),
            sy.arc.Change(type="module", path=library, kind="changed"),
        )
        self.wait_for_eq("um_fn_out", 8, timeout=5 * sy.TimeSpan.SECOND)

    def _verify_chain_edit(self, chain_a: str, chain_b: str, chain_c: str) -> None:
        task_key = self.task_key(self.arc_name)
        before = self.client.tasks.retrieve(task_key).config_hash
        self.log("Renaming the export at the bottom of the chain")
        self.overwrite_module(chain_c, CHAIN_C_SOURCE.replace("value", "renamed"))
        timeout = 10 * sy.TimeSpan.SECOND
        self.wait_for_module_status(chain_c, "success", timeout)
        for path in (chain_b, chain_a):
            self.log(f"Waiting for {path} to fail to compile")
            status = self.wait_for_module_status(path, "error", timeout)
            assert status.message == "Failed to compile", status
        self.wait_for_config_hash_change(task_key, before, timeout)
        self.log("Restoring the export with a new value")
        self._edit_and_redeploy(
            chain_c,
            CHAIN_C_SOURCE.replace("40", "50"),
            sy.arc.Change(type="module", path=chain_c, kind="changed"),
        )
        for path in (chain_b, chain_a):
            self.wait_for_module_status(path, "success", timeout)
        self.wait_for_eq("um_chain_out", 52, timeout=5 * sy.TimeSpan.SECOND)

    def _verify_automation_edit(self, source: str) -> None:
        task_key = self.task_key(self.arc_name)
        before = self.client.tasks.retrieve(task_key).config_hash
        self.log("Editing the automation's own text")
        self.overwrite_automation(
            self.arc_name, source.replace("authority 200", "authority 201")
        )
        self.wait_for_config_hash_change(task_key, before, 10 * sy.TimeSpan.SECOND)
        self._assert_changes(sy.arc.Change(type="automation", path="", kind="changed"))

    def _edit_and_redeploy(self, path: str, source: str, change: sy.arc.Change) -> None:
        """Overwrite the module, check the change the automation names, and redeploy."""
        task_key = self.task_key(self.arc_name)
        before = self.client.tasks.retrieve(task_key).config_hash
        self.overwrite_module(path, source)
        self.log("Waiting for the automation's task config hash to move")
        self.wait_for_config_hash_change(task_key, before, 10 * sy.TimeSpan.SECOND)
        self._assert_changes(change)
        self.log("Redeploying the automation")
        self.start_arc_task(self.client.tasks.retrieve(task_key))
        self.writer.write("um_start", 1)

    def _assert_changes(self, *expected: sy.arc.Change) -> None:
        """The Core stores the changes with the config hash, so no wait is needed."""
        automation = self.client.arcs.retrieve(name=self.arc_name, include_changes=True)
        assert isinstance(automation, sy.arc.Automation)
        assert automation.changes == list(expected), automation.changes

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
        self.log("Waiting for the chain stage to double it and add one")
        self.wait_for_eq("um_var_out", 85, timeout=5 * sy.TimeSpan.SECOND)
