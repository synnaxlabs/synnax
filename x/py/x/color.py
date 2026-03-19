#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re


def rgb_to_hex(rgb_str: str) -> str:
    """Convert an RGB color string (e.g., 'rgb(255, 0, 0)') to hex (e.g., '#FF0000')."""
    vals = re.findall(r"[\d.]+", rgb_str)
    r, g, b = [int(float(x)) for x in vals[:3]]
    return f"#{r:02X}{g:02X}{b:02X}"
