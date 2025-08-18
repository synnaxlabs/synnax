#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from re import S
import sys
import os
import time
from time import time as now

# Set up the path before importing framework modules
sys.path.append(os.path.join(os.path.dirname(__file__), '../..'))

from framework.TestCase import TestCase

import synnax as sy

class Bench_Latency_Response(TestCase):

    def setup(self) -> None:

        self.Expected_Timeout = 15

        self.bench_client = sy.Synnax(
            host=self.SynnaxConnection.server_address,
            port=self.SynnaxConnection.port,
            username=self.SynnaxConnection.username,
            password=self.SynnaxConnection.password,
            secure=self.SynnaxConnection.secure,
        )

        self.STATE_CHANNEL = "bench_state"
        self.CMD_CHANNEL = "bench_command"
        self.STATE = True

        time = list()

        self.bench_client.channels.create(
            name=self.STATE_CHANNEL,
            data_type=sy.DataType.UINT16,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        self.bench_client.channels.create(
            name=self.CMD_CHANNEL,
            data_type=sy.DataType.UINT16,
            virtual=True,
            retrieve_if_name_exists=True,
        )


        # Just make sure to call super() last!
        super().setup()

    def run(self) -> None:
        """
        Run the test case.
        """
        start = time.time()
        uptime = 0
        with self.bench_client.open_streamer(self.CMD_CHANNEL) as stream:
            with self.bench_client.open_writer(sy.TimeStamp.now(), self.STATE_CHANNEL) as writer:
                while uptime < 10:
                    frame = stream.read(timeout=0)
                    if frame is not None:
                        writer.write(self.STATE_CHANNEL, frame[self.CMD_CHANNEL])
                    uptime = time.time() - start


    def teardown(self) -> None:
        """
        Teardown the test case.
        """

        # Always call super() last
        super().teardown()