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
    """Request to resolve aliases to channel keys."""

    range: uuid.UUID
    aliases: list[str]


class ResolveResponse(Payload):
    """Response containing resolved aliases."""

    aliases: dict[str, ChannelKey]


class SetRequest(Payload):
    """Request to set aliases for channels."""

    range: uuid.UUID
    aliases: dict[ChannelKey, str]


class DeleteRequest(Payload):
    """Request to delete aliases."""

    range: uuid.UUID
    channels: list[ChannelKey]


class ListRequest(Payload):
    """Request to list all aliases for a range."""

    range: uuid.UUID


class ListResponse(Payload):
    """Response containing all aliases for a range."""

    aliases: dict[str, str]


class RetrieveRequest(Payload):
    """Request to retrieve aliases for specific channels."""

    range: uuid.UUID
    channels: list[ChannelKey]


class RetrieveResponse(Payload):
    """Response containing aliases for specific channels."""

    aliases: dict[str, str]


class EmptyResponse(Payload):
    """Empty response payload."""

    ...
