#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from freighter import Payload


class Authority(int):
    ABSOLUTE: Authority
    """Absolute authority. No other subject can take control of this entity while it is
    active."""
    DEFAULT: Authority


Authority.ABSOLUTE = Authority(255)
Authority.DEFAULT = Authority(1)

CrudeAuthority = int | Authority


class Subject(Payload):
    name: str
    key: str
