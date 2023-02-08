#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from enum import Enum

from synnax.exceptions import ValidationError


class FramingMode(Enum):
    KEY = "key"
    NAME = "name"
    UNOPENED = "unopened"


def open_framing_mode(
    keys: list[str] | None = None,
    names: list[str] | None = None,
) -> FramingMode:
    if keys is not None and names is not None:
        raise ValidationError(
            "keys and names cannot both be specified when calling open()"
        )
    elif keys is None and names is None:
        raise ValidationError("keys or names must be specified when calling open()")
    return FramingMode.KEY if keys is not None else FramingMode.NAME
