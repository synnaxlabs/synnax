#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from synnax.channel import (
    ChannelKeys,
    ChannelNames,
    ChannelKey,
    ChannelName,
    ChannelParams,
)
from synnax.ranger.payload import RangePayload
from synnax.telem import TimeRange, Series
from synnax.framer import FrameClient, Frame


class Range(RangePayload):
    __frame_client: FrameClient | None

    def __init__(
        self,
        name: str,
        time_range: TimeRange,
        key: str = "",
        frame_client: FrameClient = None,
    ):
        super().__init__(
            name=name,
            time_range=time_range,
            key=key,
        )
        self.__frame_client = frame_client

    @overload
    def read(self, params: ChannelKey | ChannelName) -> Series:
        ...

    @overload
    def read(self, params: ChannelKeys | ChannelNames) -> Frame:
        ...

    def read(
        self,
        params: ChannelParams,
    ) -> Series | Frame:
        return self.__frame_client.read(
            self.time_range,
            params,
        )

    def to_payload(self) -> RangePayload:
        return RangePayload(
            name=self.name,
            time_range=self.time_range,
            key=self.key,
        )
