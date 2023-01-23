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

from .payload import ChannelPayload
from .retrieve import ChannelRetriever


class ChannelRegistry:
    retriever: ChannelRetriever
    channels: dict[str, ChannelPayload]

    def __init__(self, retriever: ChannelRetriever, channels=None) -> None:
        if channels is None:
            channels = list()
        self.retriever = retriever
        self.channels = {ch.name: ch for ch in channels}

    def get(self, key: str) -> ChannelPayload | None:
        record = self.channels.get(key, None)
        if record is None:
            record = self.retriever.retrieve(key=key)
            self.channels[key] = record
        return record

    def get_n(self, keys: list[str]) -> list[ChannelPayload]:
        results = list()
        retrieve_keys = list()
        for key in keys:
            record = self.channels.get(key, None)
            if record is not None:
                results.append(record)
            retrieve_keys.append(key)
        if retrieve_keys:
            results.extend(self.retriever.filter(keys=retrieve_keys))
        return results
