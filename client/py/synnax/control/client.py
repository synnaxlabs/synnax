#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.channel.payload import ChannelParams
from synnax.channel.retrieve import ChannelRetriever
from synnax.control.controller import Controller
from synnax.framer import Client as FrameClient
from synnax.telem.control import Authority, CrudeAuthority


class Client:
    framer: FrameClient
    retriever: ChannelRetriever

    def __init__(self, framer: FrameClient, channels: ChannelRetriever) -> None:
        self.framer = framer
        self.retriever = channels

    def acquire(
        self,
        name: str,
        read: ChannelParams,
        write: ChannelParams,
        write_authorities: CrudeAuthority | list[CrudeAuthority] = Authority.ABSOLUTE,
    ) -> Controller:
        return Controller(
            name=name,
            write=write,
            read=read,
            frame_client=self.framer,
            retriever=self.retriever,
            write_authorities=write_authorities,
        )
