#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import Payload


class Rack(Payload):
    key: int = 0
    name: str = ""


class Task(Payload):
    key: int = 0
    name: str = ""
    type: str = ""
    config: str = ""


class Device(Payload):
    key: str = ""
    location: str = ""
    rack: int = 0
    name: str = ""
    make: str = ""
    model: str = ""
    properties: str = ""
