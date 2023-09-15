#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
 
from freighter import StreamClient
from synnax import framer
from synnax.control.controller import Controller
from synnax.channel import ChannelParams, ChannelRetriever

class Client:
    framer: framer.Client
    retriever: ChannelRetriever

    def __init__(self, framer: framer.Client, channels: ChannelRetriever) -> None:
        self.framer = framer
        self.retriever = channels

    def acquire(
        self,
        read: ChannelParams,
        write: ChannelParams,
    ) -> Controller:
        return Controller(
            write=write,
            read=read,
            framer=self.framer,
            retriever=self.retriever,
        )


