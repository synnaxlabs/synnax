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
    """Joins where and flat with an and. A flat tree with no lines adds nothing.

    :param where: The tree the caller passed, or None.
    :param flat: The tree a client builds from the keyword arguments of a retrieve.
        Its type must declare an ``and_`` field.
    :returns: The joined tree.
    """
    if where is None:
        return flat
    if flat == type(flat)():
        return where
    return type(flat)(and_=[where, flat])
