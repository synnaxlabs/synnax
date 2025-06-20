#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Generic, Literal, TypeVar
from uuid import uuid4

from freighter import Payload
from pydantic import Field

from synnax.telem import TimeStamp

SUCCESS_VARIANT = "success"
INFO_VARIANT = "info"
WARNING_VARIANT = "warning"
ERROR_VARIANT = "error"
DISABLED_VARIANT = "disabled"
LOADING_VARIANT = "loading"

Variant = Literal[
    "success",
    "info",
    "warning",
    "error",
    "disabled",
    "loading",
]
"""Represents the variant of a status message."""

D = TypeVar("D", bound=Payload, default=Payload)


class Status(Payload, Generic[D]):
    key: str = Field(default=str(uuid4()))
    variant: Variant
    message: str
    description: str = ""
    time: TimeStamp = Field(default=TimeStamp.now())
    details: D
