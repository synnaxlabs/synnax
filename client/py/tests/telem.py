#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy


def seconds_linspace(start: int, count: int) -> list[sy.TimeSpan]:
    """Generates a list of TimeSpan values from start to start + count"""
    return [start * sy.TimeSpan.SECOND + i * sy.TimeSpan.SECOND for i in range(count)]
