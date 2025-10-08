#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
import sys
from pathlib import Path

import synnax as sy

from framework.test_case import TestCase


class OPCUASimServer(TestCase):
    """
    Run the OPC UA simulated server
    """

    def run(self) -> None:
        """
        Run the test case.
        """
        # Navigate from integration/tests/driver/ up to synnax root, then to driver/opc/dev/
        integration_dir = Path(__file__).resolve().parent.parent.parent
        synnax_root = integration_dir.parent
        server_script = synnax_root / "driver" / "opc" / "dev" / "server_extended.py"

        try:
            subprocess.run(
                [sys.executable, str(server_script)], timeout=30, check=True
            )
        except subprocess.TimeoutExpired:
            pass
