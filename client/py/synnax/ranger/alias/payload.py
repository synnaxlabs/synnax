#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

from freighter import Payload

from synnax.channel import ChannelKey


class ResolveRequest(Payload):
    range: uuid.UUID
    aliases: list[str]


class ResolveResponse(Payload):
    aliases: dict[str, ChannelKey]


class SetRequest(Payload):
    range: uuid.UUID
    aliases: dict[ChannelKey, str]


class EmptyResponse(Payload): ...
