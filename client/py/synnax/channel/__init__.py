#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.channel.client import Channel, Client
from synnax.channel.payload import (
    Key,
    NormalizedKeyResult,
    NormalizedNameResult,
    Operation,
    Params,
    Payload,
    has_params,
    normalize_params,
)
from synnax.channel.retrieve import (
    CacheRetriever,
    ClusterRetriever,
    Retriever,
    retrieve_required,
)
from synnax.channel.writer import Writer
from synnax.util.deprecation import deprecated_getattr

_DEPRECATED = {
    "ChannelClient": "Client",
    "ChannelKey": "Key",
    "ChannelParams": "Params",
    "ChannelPayload": "Payload",
    "ChannelRetriever": "Retriever",
    "normalize_channel_params": "normalize_params",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "Channel",
    "Client",
    "Key",
    "Operation",
    "Params",
    "Payload",
    "normalize_params",
    "CacheRetriever",
    "ClusterRetriever",
    "Retriever",
    "retrieve_required",
    "Writer",
    "has_params",
    "NormalizedNameResult",
    "NormalizedKeyResult",
]
