#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import Instrumentation, trace
from freighter import (
    Payload,
    UnaryClient,
    send_required,
)
from synnax.channel.payload import (
    ChannelKeys,
    ChannelNames,
    ChannelParams,
    normalize_channel_params,
)
from synnax.telem import TimeRange


class _Request(Payload):
    keys: ChannelKeys | None
    names: ChannelNames | None
    bounds: TimeRange


class _Response(Payload):
    ...


_ENDPOINT = "/frame/delete"


class Deleter:
    """
    Deleter is used to delete a time range of telemetry from the data engine.
    """

    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(self, client: UnaryClient, instrumentation: Instrumentation) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def delete(
        self,
        channels: ChannelParams,
        tr: TimeRange,
    ) -> None:
        normal = normalize_channel_params(channels)
        req = _Request(
            **{
                normal.variant: normal.channels,
                "bounds": tr,
            }
        )
        send_required(self._client, _ENDPOINT, req, _Response)
