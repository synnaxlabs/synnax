#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

from freighter import Payload

from synnax.exceptions import ValidationError
from synnax.util.primitive import is_primitive


class Pair(Payload):
    range: uuid.UUID
    key: str
    value: str

    def __init__(self, **kwargs):
        value = kwargs.get("value")
        if not isinstance(value, str):
            if not is_primitive(value) and type(value).__str__ == object.__str__:
                raise ValidationError(f"""
                Synnax has no way of casting {value} to a string when setting metadata
                on a range. Please convert the value to a string before setting it.
                """)
        kwargs["value"] = str(value)
        super().__init__(**kwargs)
