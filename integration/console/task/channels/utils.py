#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


def is_numeric_string(value: str | bool) -> bool:
    """Check if a string represents a numeric value."""
    if not isinstance(value, str):
        return False

    value = value.strip()
    if not value:
        return False

    if value.startswith("-"):
        value = value[1:]

    parts = value.split(".")
    if len(parts) > 2:
        return False

    return all(part.isdigit() for part in parts if part)
