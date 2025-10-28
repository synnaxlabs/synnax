#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import numpy as np
import synnax as sy

from tests.latency.latency import Latency


class Abcd(Latency):
    """

    Testing the general, periodic latency of 3 async processes.

    Each process is running at a consistent 100 Hz, to emulate some real-world
    automation. Each process will create its own "time" channel, of which the
    sibling process will subscribe to. The subscribers will then output a
    corresponding "follower" timestamp channel at the current time.

    This will allow us to measure the time it takes for data to posted to
    Synnax and how long another process was able to act on it. Since ABC will
    be running at a consistent 100 Hz, we will expect at least some latency
    due to any phase offsets.

    A > T_0 = time.time()
    B > T_1 = read_tlm(T_0)
    C > T_2 = read_tlm(T_1)
    A > T_3 = read_tlm(T_2)


    The latency report is only valid if the processes are running on the same
    machine OR you have enabled synchronization between your servers.


    One final note, the mechanism for which the test framework reads and writes
    tlm will introduce a fair bit of stacked latency between T_0 and T_3.

    These files were meant to be simple to use, but you may want to rewrite
    the Synnax client connection to better represent your system.

    """

    def setup(self) -> None:
        super().setup()
        """
        Setup the test case.
        """
        self.log("WARNING (⚠️): This test does not have any reporting.")
        self.configure(loop_rate=0.01, manual_timeout=20)

        # Get mode from matrix parameters
        self.mode = self.params.get("mode")
        if self.mode is None:
            raise ValueError("Missing required parameter 'mode' from matrix")

        if self.mode == "a":
            self.add_channel(
                name="t_a",
                data_type=sy.DataType.TIMESTAMP,
                initial_value=sy.TimeStamp.now(),
                append_name=False,
            )
            self.add_channel(
                name="t_d",
                data_type=sy.DataType.TIMESTAMP,
                initial_value=sy.TimeStamp.now(),
                append_name=False,
            )
            self.subscribe(["t_c"])

        elif self.mode == "b":
            self.add_channel(
                name="t_b",
                data_type=sy.DataType.TIMESTAMP,
                initial_value=sy.TimeStamp.now(),
                append_name=False,
            )
            self.subscribe("t_a")

        elif self.mode == "c":
            self.add_channel(
                name="t_c",
                data_type=sy.DataType.TIMESTAMP,
                initial_value=sy.TimeStamp.now(),
                append_name=False,
            )
            self.subscribe("t_b")

        if self.mode == "d":
            self.add_channel("d_ab", sy.DataType.FLOAT64, 0, False)
            self.add_channel("d_bc", sy.DataType.FLOAT64, 0, False)
            self.add_channel("d_cd", sy.DataType.FLOAT64, 0, False)
            self.add_channel("d_da", sy.DataType.FLOAT64, 0, False)
            self.subscribe(["t_a", "t_b", "t_c", "t_d"])

    def run(self) -> None:
        """
        Run the test case.
        """
        self.log("Starting run()")
        if self.mode == "a":
            while self.loop.wait() and self.should_continue:
                td = self.read_tlm("t_c", None)
                if td is not None:
                    self.write_tlm("t_d", td)

                self.write_tlm("t_a", sy.TimeStamp.now())

        elif self.mode == "b":
            while self.loop.wait() and self.should_continue:
                t_b = self.read_tlm("t_a", None)
                if t_b is not None:
                    self.write_tlm("t_b", t_b)

        elif self.mode == "c":
            while self.loop.wait() and self.should_continue:
                t_c = self.read_tlm("t_b", None)
                if t_c is not None:
                    self.write_tlm("t_c", t_c)

        elif self.mode == "d":
            # 100Hz for 20 seconds
            delta_a_b = np.zeros(1000 * self._manual_timeout)
            delta_b_c = np.zeros(1000 * self._manual_timeout)
            delta_c_d = np.zeros(1000 * self._manual_timeout)
            delta_d_a = np.zeros(1000 * self._manual_timeout)
            idx = 0

            while self.loop.wait() and self.should_continue:
                # Just assume we'll never exceed
                # the 20 second limit for the np arrays
                t_a = self.read_tlm("t_a")
                t_b = self.read_tlm("t_b")
                t_c = self.read_tlm("t_c")
                t_d = self.read_tlm("t_d")

                assert (
                    t_a is not None
                    and t_b is not None
                    and t_c is not None
                    and t_d is not None
                )

                d_ab = (t_a - t_b) / 1e9
                d_bc = (t_b - t_c) / 1e9
                d_cd = (t_c - t_d) / 1e9
                d_da = -(t_d - t_a) / 1e9
                self.write_tlm("d_ab", d_ab)
                self.write_tlm("d_bc", d_bc)
                self.write_tlm("d_cd", d_cd)
                self.write_tlm("d_da", d_da)

                delta_a_b[idx] = d_ab
                delta_b_c[idx] = d_bc
                delta_c_d[idx] = d_cd
                delta_d_a[idx] = d_da

                idx += 1

            self.log("WARNING (⚠️): LatencyABC Report not implemented...")
