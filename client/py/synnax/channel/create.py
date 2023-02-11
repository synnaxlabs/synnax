#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import HTTPClientPool, Payload, UnaryClient

from synnax.channel.payload import ChannelPayload


class _Request(Payload):
    channels: list[ChannelPayload]


class _Response(Payload):
    channels: list[ChannelPayload]


class ChannelCreator:
    _ENDPOINT = "/channel/create"
    client: UnaryClient

    def __init__(self, client: HTTPClientPool):
        self.client = client.post_client()

    def create(
        self,
        channels: list[ChannelPayload],
    ) -> list[ChannelPayload]:
        req = _Request(channels=channels)
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        return res.channels
