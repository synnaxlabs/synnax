#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
import os
import time
import numpy as np


# Set up the path before importing framework modules
sys.path.append(os.path.join(os.path.dirname(__file__), '..'))
from framework.TestCase import TestCase


import synnax as sy


class Latency_ABC(TestCase):

    """
    
    Testing the general, periodic latency of 3 async processes.

    Each process is running at a conisitent 100Hz, to emulate some real-world 
    automation. Each process will create its own "time" channel, of which the 
    sibling proceess will subscribe to. The subsribers will then output a 
    corresponding "follower" timstamp channel at the current time.

    This will allow us to measure the time it takes for data to posted to 
    Synnax and how long another process was able to act on it. Since ABC will 
    be running at a consitent 100Hz, we will expect at least some latency 
    due to any phase offsets. 

    A > T_0 = time.time()
    B > T_1 = get_latest(T_0)
    C > T_2 = get_latest(T_1)
    A > T_3 = get_latest(T_2)


    The latency report is only valid if the proceses are running on the same 
    machine OR you have enabled synchronization between your servers.


    One final note, the mechanism for which the test framework reads and writes
    tlm will introduce a fair bit of stacked latency between T_0 and T_3.

    These files were meant to be simple to use, but you may want to rewrite
    the Synnax client connection to better represent your system.
    
    """
    

    def setup(self) -> None:
        """
        Setup the test case.
        """

        self.configure(
            loop_rate=0.01,
            manual_timeout=30
        )
        

        self.mode = self.name[-1] # A, B, , 
        
        if self.mode == "a":
            self.add_channel(name="t_a", data_type=sy.DataType.TIMESTAMP, initial_value=sy.TimeStamp.now(), append_name=False),
            self.add_channel(name="t_d", data_type=sy.DataType.TIMESTAMP, initial_value=sy.TimeStamp.now(), append_name=False)
            self.subscribe(["t_c"])

        elif self.mode == "b":
            self.add_channel(name="t_b", data_type=sy.DataType.TIMESTAMP, initial_value=sy.TimeStamp.now(), append_name=False)
            self.subscribe("t_a")

        elif self.mode == "c":
            self.add_channel(name="t_c", data_type=sy.DataType.TIMESTAMP, initial_value=sy.TimeStamp.now(), append_name=False)
            self.subscribe("t_b")

        if self.mode == "d":
            self.subscribe(["t_a", "t_b", "t_c", "t_d"])
            self.add_channel("d_ab",sy.DataType.FLOAT64, 0, False)
            self.add_channel("d_bc",sy.DataType.FLOAT64, 0, False)
            self.add_channel("d_cd",sy.DataType.FLOAT64, 0, False)
            self.add_channel("d_da",sy.DataType.FLOAT64, 0, False)


    def run(self) -> None:
        """
        Run the test case.
        """
        
        # Wait for the client thread to start and populate data
        if self.mode == "a":
            while self.loop.wait() and self.should_continue:
                td = self.read_tlm("t_c", None)
                if td is not None:
                    self.write_tlm('t_d', td)
                
                self.write_tlm('t_a', sy.TimeStamp.now())           
        
        elif self.mode == "b":
            while self.loop.wait() and self.should_continue:
                t_b = self.read_tlm("t_a", None)
                if t_b is not None:
                    self.write_tlm('t_b', t_b)

        elif self.mode == "c":
            while self.loop.wait() and self.should_continue:
                t_c = self.read_tlm("t_b", None)
                if t_c is not None:
                    self.write_tlm('t_c', t_c)

        elif self.mode == "d":
            # 100Hz for 20 seconds
            delta_a_b = np.zeros(1000*20) 
            delta_b_c = np.zeros(1000*20)
            delta_c_d = np.zeros(1000*20)
            delta_d_a = np.zeros(1000*20)
            idx = 0

            time.sleep(5) # Let other processes start
            while self.loop.wait() and self.should_continue:

                    # Just assume we'll never exceed
                    # the 20 second limit for the np arrays
                    t_a = self.read_tlm("t_a")
                    t_b = self.read_tlm("t_b")
                    t_c = self.read_tlm("t_c")
                    t_d = self.read_tlm("t_d")

                    d_ab = (t_a - t_b)/1E9
                    d_bc = (t_b - t_c)/1E9
                    d_cd = (t_c - t_d)/1E9
                    d_da = -(t_d - t_a)/1E9
                    self.write_tlm("d_ab", d_ab)
                    self.write_tlm("d_bc", d_bc)
                    self.write_tlm("d_cd", d_cd)
                    self.write_tlm("d_da", d_da)


                    delta_a_b[idx] = d_ab
                    delta_b_c[idx] = d_bc
                    delta_c_d[idx] = d_cd
                    delta_d_a[idx] = d_da
                    
                    idx += 1
            
        
    def teardown(self) -> None:
        """`
        Teardown the test case.
        """

        # Always call super() last
        super().teardown()