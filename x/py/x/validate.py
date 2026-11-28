#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Checks for pydantic AfterValidator that pydantic's own constraints lack."""

from uuid import UUID


def non_nil_uuid(value: UUID) -> UUID:
    """Refuses the nil UUID, which a required UUID field cannot hold."""
    if value.int == 0:
        raise ValueError("must be non-zero")
    return value
