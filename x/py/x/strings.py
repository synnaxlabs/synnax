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

import numpy as np


def validate_and_sanitize_name(name: str) -> str:
    """Sanitize name to contain only alphanumeric characters, hyphens, and underscores."""
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


def format_float(v: float | np.floating) -> str:
    """Format a float as the shortest text that parses back to it at its own width.

    The text is in exponent notation below 1e-6 or from 1e21, with an unpadded exponent
    (1e-7, 1e+21). NaN is NaN, and infinities are Infinity and -Infinity. It matches the
    float text of every Synnax runtime.

    :param v: the float to format. A np.float32 is formatted at 32-bit precision.
    :returns: the text of v.
    """
    if np.isnan(v):
        return "NaN"
    if np.isinf(v):
        return "Infinity" if v > 0 else "-Infinity"
    f = v if isinstance(v, np.floating) else np.float64(v)
    a = abs(f)
    if a != 0 and (a < 1e-6 or a >= 1e21):
        return np.format_float_scientific(f, unique=True, trim="-", exp_digits=1)
    return np.format_float_positional(f, unique=True, trim="-")
