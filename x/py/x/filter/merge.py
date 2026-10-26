#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel


def merge[W: BaseModel](where: W | None, flat: W) -> W:
    """Returns where and flat joined by and. A flat tree with no lines adds nothing.

    Clients build flat from the keyword arguments of a retrieve.
    """
    if where is None:
        return flat
    if flat == type(flat)():
        return where
    return type(flat)(and_=[where, flat])
