#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Arc programs on the real-time C++ runtime of the cRIO's Driver. Neither test needs
C Series modules."""

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channel
from tests.arc.arc import ArcCase
from tests.arc.wait_accuracy_samples import _WaitSamples
from tests.hitl.crio_case import CRIOCase

DOUBLE_SOURCE = """
func double() {x f64} f64 {
    return x * 2.0
}
crio_arc_in -> double() -> crio_arc_out
"""

# Over the error of setting the clock through SSH, and far under a broken sync.
_MAX_CLOCK_SKEW = sy.TimeSpan.SECOND


class _ArcOnCRIO(CRIOCase, ArcCase):
    """Deploys the Arc to the rack of the Driver that setup started on the cRIO."""

    def _retrieve_rack(self) -> None:
        self.log(f"Selecting rack: {self.rack.name} (key: {self.rack.key})")


class CRIOArc(_ArcOnCRIO):
    """Doubles each value written to an input channel. The output stamps come from the
    cRIO's clock, so they must agree with this machine's clock."""

    arc_source = DOUBLE_SOURCE
    arc_name_prefix = "CRIOArc"
    start_cmd_channel = "crio_arc_start_cmd"
    subscribe_channels = ["crio_arc_out"]

    def setup(self) -> None:
        create_virtual_channel(
            self.client, "crio_arc_in", data_type=sy.DataType.FLOAT64
        )
        create_indexed_pair(self.client, "crio_arc_out", data_type=sy.DataType.FLOAT64)
        super().setup()

    def verify_sequence_execution(self) -> None:
        for value in [21.0, -1.5]:
            self.writer.write("crio_arc_in", value)
            self.wait_for_eq("crio_arc_out", value * 2)
        now = sy.TimeStamp.now()
        stamp = sy.TimeStamp(
            self.client.read_latest("crio_arc_out_time").to_numpy()[-1]
        )
        skew = sy.TimeSpan(abs(int(now) - int(stamp)))
        if skew > _MAX_CLOCK_SKEW:
            self.fail(f"The cRIO's last output is {skew} off this machine's clock")


class CRIOWaitSamples(_ArcOnCRIO, _WaitSamples):
    """Measures the waits of a sequence on the cRIO's real-time runtime."""

    profile_ms = [20, 10, 5, 1]
    driver_os = "NI Linux RT"
