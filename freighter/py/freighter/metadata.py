#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass
from typing import MutableMapping, Any


@dataclass
class MetaData:
    """MetaData is the metadata associated with a freighter transport request.

    :param protocol: The protocol to use for the transport request.
    :param target: The target to use for the transport request.
    """

    protocol: str
    """Protocol used to issue the request."""

    target: str
    """Target is the target of the request."""

    params: MutableMapping[str, str]
    """Arbitrary string parameters that can be set by clinet side middleware
    and read by server side middleware"""

    def __init__(self, protocol: str, target: str):
        self.protocol = protocol
        self.target = target
        self.params = {}

    def set(self, key: str, value: str) -> None:
        self.params[key] = value

    def get(self, key: str, default: Any = None) -> Any:
        return self.params.get(key, default)
