#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TypedDict


class Replace[T](TypedDict):
    """Replaces a struct field of an update whole.

    Attributes:
        set: Is the new value of the field.
    """

    set: T


# Struct is the change that an update makes to a struct field: the partial U merges into
# the field, and a Replace replaces it. An optional struct field takes a Change of a
# Struct, whose None clears the field.
type Struct[U, T] = U | Replace[T]

# Change is the change that an update makes to a field of a TypedDict partial: a value
# sets the field, and None clears it. An absent key makes no change.
type Change[T] = T | None
