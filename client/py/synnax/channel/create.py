#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import HTTPClientFactory, Payload, UnaryClient

from ..telem import DATA_TYPE_UNKNOWN, Rate, UnparsedDataType, UnparsedRate
from .payload import ChannelPayload


class _Request(Payload):
    channels: list[ChannelPayload]


class _Response(Payload):
    channels: list[ChannelPayload]


class ChannelCreator:
    _ENDPOINT = "/channel/create"
    client: UnaryClient

    def __init__(self, client: HTTPClientFactory):
        self.client = client.post_client()

    def create(
        self,
        name: str = "",
        node_id: int = 0,
        rate: UnparsedRate = Rate(0),
        data_type: UnparsedDataType = DATA_TYPE_UNKNOWN,
        index: str = "",
        is_index: bool = False,
    ) -> ChannelPayload:
        return self.create_many(
            [
                ChannelPayload(
                    data_type=data_type,
                    name=name,
                    node_id=node_id,
                    rate=rate,
                    index=index,
                    is_index=is_index,
                )
            ]
        )[0]

    def create_many(self, channels: list[ChannelPayload]) -> list[ChannelPayload]:
        req = _Request(channels=channels)
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        return res.channels
