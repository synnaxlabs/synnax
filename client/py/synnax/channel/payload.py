#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import Payload

from synnax.telem import DataType, Rate


ChannelKey = int
ChannelName = str
ChannelKeys = tuple[int] | list[int]
ChannelNames = tuple[str] | list[str]
ChannelParams = ChannelKeys | ChannelNames | ChannelKey | ChannelName


class ChannelPayload(Payload):
    """A payload container that represent the properties of a channel exchanged to and
    from the Synnax server.
    """

    key: ChannelKey = 0
    data_type: DataType
    rate: Rate = Rate(0)
    name: str = ""
    leaseholder: int = 0
    is_index: bool = False
    index: ChannelKey = 0
