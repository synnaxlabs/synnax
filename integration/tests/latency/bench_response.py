#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from re import S
from time import time as now

import synnax as sy

from tests.latency.latency import Latency


class BenchResponse(Latency):

    def setup(self) -> None:
        super().setup()
        self.set_manual_timeout(10)

        self.bench_client = sy.Synnax(
            host=self.synnax_connection.server_address,
            port=self.synnax_connection.port,
            username=self.synnax_connection.username,
            password=self.synnax_connection.password,
            secure=self.synnax_connection.secure,
        )

        self.state_channel = "bench_state"
        self.cmd_channel = "bench_command"

        self.bench_client.channels.create(
            name=self.state_channel,
            data_type=sy.DataType.UINT16,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        self.bench_client.channels.create(
            name=self.cmd_channel,
            data_type=sy.DataType.UINT16,
            virtual=True,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        """
        Run the test case.
        """

        state_channel: str = self.state_channel
        cmd_channel: str = self.cmd_channel

        try:
            with self.bench_client.open_streamer(cmd_channel) as stream:
                with self.bench_client.open_writer(
                    sy.TimeStamp.now(), state_channel
                ) as writer:
                    while self.should_continue:
                        frame = stream.read(timeout=3)
                        if frame is not None:
                            writer.write(state_channel, frame[cmd_channel])

        except Exception as e:
            raise Exception(f"EXCEPTION: {e}")
