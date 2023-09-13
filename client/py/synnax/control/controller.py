#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.framer import Writer, Streamer
from synnax.channel import ChannelKey, ChannelName, ChannelRetriever
from synnax.telem import TimeStamp

class Controller:
    writer: Writer
    streamer: Streamer
    idx_map: dict[ChannelKey, ChannelKey]
    retriever: ChannelRetriever

    def set(self, ch: ChannelKey | ChannelName, value: int | float):
        ch = self.retriever.retrieve(ch)
        idx = self.retriever.retrieve(self.idx_map[ch.key])

        self.writer.write({
            [ch.key]: value,
            [idx.key]: TimeStamp.now(),
        })


    