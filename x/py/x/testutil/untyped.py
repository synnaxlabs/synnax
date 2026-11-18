#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable


def untyped[R](fn: Callable[..., R]) -> Callable[..., R]:
    """Return fn with its parameters widened to accept any arguments.

    Tests use it to call fn the way an untyped caller can: with deprecated
    positional arguments or invalid values that the static signature rejects. The
    return type stays the same.
    """
    return fn
