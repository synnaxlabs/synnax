#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import re
import string

from pydantic import AfterValidator


def validate_and_sanitize_name(name: str) -> str:
    """Sanitize name to contain only alphanumeric characters, hyphens, and
    underscores.
    """
    sanitized = re.sub(r"[^a-zA-Z0-9_-]", "", name)

    if not sanitized:
        raise ValueError("Name must contain at least one alphanumeric character")

    sanitized = sanitized.strip("_-")
    if not sanitized:
        raise ValueError("Name cannot consist only of hyphens and underscores")

    return sanitized


def random_name() -> str:
    """Get a random name, which is a random 15-character alphabetic string."""
    return "".join(random.choice(string.ascii_letters) for _ in range(15))


def utf16_length(value: str) -> int:
    """Returns the length of value in UTF-16 code units, which matches JavaScript
    string length. An emoji counts as two units.
    """
    return len(value.encode("utf-16-le", "surrogatepass")) // 2


def min_length(limit: int) -> AfterValidator:
    """Returns a validator that rejects a string shorter than limit UTF-16 code
    units.
    """

    def check(value: str) -> str:
        if utf16_length(value) < limit:
            raise ValueError(f"must be at least {limit} characters long")
        return value

    return AfterValidator(check)


def max_length(limit: int) -> AfterValidator:
    """Returns a validator that rejects a string longer than limit UTF-16 code
    units.
    """

    def check(value: str) -> str:
        if utf16_length(value) > limit:
            raise ValueError(f"must be at most {limit} characters long")
        return value

    return AfterValidator(check)
