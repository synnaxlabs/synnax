#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reads the filter lines that a partial record writes as a plain value or a list."""

from typing import Any


def read_set(value: Any, operator: type) -> Any:
    """Reads a Set, Ordered, or String operator. A plain value is ``eq``, and a list is
    ``in``. A dict or an operator passes through as it is."""
    if value is None or isinstance(value, (dict, operator)):
        return value
    if isinstance(value, (list, tuple)):
        return {"in": list(value)}
    return {"eq": value}


def read_bool(value: Any, operator: type) -> Any:
    """Reads a Bool operator. A plain boolean is ``eq``."""
    if isinstance(value, bool):
        return {"eq": value}
    return value


def read_contains(value: Any, operator: type) -> Any:
    """Reads a Contains operator. A plain value or a list matches the records that
    hold any of them."""
    if value is None or isinstance(value, (dict, operator)):
        return value
    if isinstance(value, (list, tuple)):
        return {"any": list(value)}
    return {"any": [value]}


def read_list(value: Any) -> Any:
    """Reads a typed line that takes a list. A plain value is a list of one."""
    if value is None or isinstance(value, (list, tuple)):
        return value
    return [value]
