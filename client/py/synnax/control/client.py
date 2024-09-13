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
    """Central client for control related tasks in the Synnax client, primarily used for
    acquiring control to run sequences.
    """

    framer: FrameClient
    retriever: ChannelRetriever

    def __init__(self, framer: FrameClient, channels: ChannelRetriever) -> None:
        self.framer = framer
        self.retriever = channels

    def acquire(
        self,
        name: str,
        read: ChannelParams | None,
        write: ChannelParams | None,
        write_authorities: CrudeAuthority | list[CrudeAuthority] = Authority.ABSOLUTE,
    ) -> Controller:
        """Opens a new controller for executing control sequences. We recommend using
        this method under a context manager to ensure that the controller is properly
        closed after use.

        :param name: A human-readable name for the controller to identify it across
        Synnax.
        :param read: A list of channels that the controller will need to read from in
        order to execute its sequences. Any channels not on this list will not be
        available to make control decisions.
        :param write: A list of channels that the controller will need to write to
        during operation. Any channels not on this list will not be writable by the
        controller.
        :param write_authorities: A single or list of authorities that the controller
        defines when writing to the channels. This can be a single authority or a list
        of authorities corresponding to the channels passed in the write parameter.
        """
        return Controller(
            name=name,
            write=write,
            read=read,
            frame_client=self.framer,
            retriever=self.retriever,
            write_authorities=write_authorities,
        )
