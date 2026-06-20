#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


def none_to_empty[K, V](v: dict[K, V] | None) -> dict[K, V]:
    """Return v unchanged, or an empty dict if v is None."""
    return dict() if v is None else v
