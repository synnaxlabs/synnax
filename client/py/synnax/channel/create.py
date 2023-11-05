#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import Instrumentation, trace
from freighter import Payload, UnaryClient

from synnax.channel.payload import ChannelPayload


class _Request(Payload):
    channels: list[ChannelPayload]


_Response = _Request


class ChannelCreator:
    __ENDPOINT = "/channel/create"
    __client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation,
    ):
        self.__client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def create(
        self,
        channels: list[ChannelPayload],
    ) -> list[ChannelPayload]:
        req = _Request(channels=channels)
        res, exc = self.__client.send(self.__ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        return res.channels
