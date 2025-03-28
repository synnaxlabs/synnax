#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from datetime import datetime
from typing import Any

from .ext import ExtType, Timestamp

def packb(o: Any, **kwargs: Any) -> Any | None: ...
def unpackb(
    packed: Any, **kwargs: Any
) -> (
    int
    | Any
    | list[Any]
    | tuple[Any, ...]
    | dict[Any, Any]
    | bytes
    | str
    | float
    | datetime
    | Timestamp
    | ExtType
    | bytearray
    | bool
    | None
): ...
