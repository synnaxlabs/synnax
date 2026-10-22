#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import enum
from dataclasses import dataclass
from typing import TypeAlias

from ..pdu.device import ModbusDeviceIdentification

SimValueType: TypeAlias = (
    int
    | float
    | str
    | bytes
    | list[int]
    | list[float]
    | list[str]
    | list[bytes]
    | list[bool]
)

class DataType(enum.IntEnum):
    INVALID = 1
    BITS = ...
    REGISTERS = ...

@dataclass
class SimData:
    address: int
    count: int = 1
    values: SimValueType = 0
    datatype: DataType = DataType.INVALID
    string_encoding: str = "utf-8"
    readonly: bool = False

@dataclass
class SimDevice:
    id: int
    simdata: (
        SimData
        | list[SimData]
        | tuple[list[SimData], list[SimData], list[SimData], list[SimData]]
    )
    use_bit_addressing: bool | None = None
    identity: ModbusDeviceIdentification | None = None
