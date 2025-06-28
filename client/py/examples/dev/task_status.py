#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

<<<<<<<< HEAD:client/py/synnax/status/status.py
from typing import Generic, Literal, TypeVar
from uuid import uuid4

from freighter import Payload
from pydantic import Field

from synnax.telem import TimeStamp
========
import synnax as sy
>>>>>>>> d0e4900de158a437b73160b89b1f0eaf95b4254e:client/py/examples/dev/task_status.py

client = sy.Synnax(
    host="localhost",
    port=9090,
    secure=False,
    username="synnax",
    password="seldon",
)

<<<<<<<< HEAD:client/py/synnax/status/status.py
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
========

with client.open_streamer(["sy_device_status"]) as s:
    for frame in s:
        for v in frame["sy_device_status"]:
            print(v)
>>>>>>>> d0e4900de158a437b73160b89b1f0eaf95b4254e:client/py/examples/dev/task_status.py
