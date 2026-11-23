#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from synnax.color.types_gen import FixedRef, NamedRef, Ref
from x.color import Color, Crude


def to_ref(color: Crude | Ref) -> Ref:
    """Returns the color as a reference: a reference as it is, and any other color as a
    fixed color.

    :param color: A reference, or a color such as ``"#FF0000"``.
    :raises ValueError: If the color is not a valid color.
    """
    if isinstance(color, NamedRef | FixedRef):
        return color
    return FixedRef(value=Color(color))
