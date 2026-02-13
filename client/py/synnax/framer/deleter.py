#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import Instrumentation, trace
from freighter import UnaryClient, send_required
from pydantic import BaseModel

import synnax.channel.payload as channel
from synnax.telem import TimeRange


class _Request(BaseModel):
    keys: list[channel.Key] | tuple[channel.Key] | None = None
    names: list[str] | tuple[str] | None = None
    bounds: TimeRange


class _Response(BaseModel): ...


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
        channels: channel.Params,
        tr: TimeRange,
    ) -> None:
        normal = channel.normalize_params(channels)
        req = _Request(
            **{
                normal.variant: normal.channels,
                "bounds": tr,
            }
        )
        send_required(self._client, "/frame/delete", req, _Response)
