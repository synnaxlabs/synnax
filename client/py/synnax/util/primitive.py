#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

Primitive = str | int | float | bool | None


def is_primitive(value: object) -> bool:
    return isinstance(value, (str, int, float, bool, type(None)))
