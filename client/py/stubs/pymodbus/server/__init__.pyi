#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable

from ..datastore import ModbusServerContext
from ..pdu.device import ModbusDeviceIdentification
from ..simulator import SimDevice
from ..simulator.simcore import SimCore

class ModbusTcpServer:
    context: ModbusServerContext | SimCore
    def __init__(
        self,
        context: ModbusServerContext | SimDevice | list[SimDevice],
        *,
        identity: ModbusDeviceIdentification | None = None,
        address: tuple[str, int] = ("", 502),
        ignore_missing_devices: bool = False,
        broadcast_enable: bool = False,
        trace_connect: Callable[[bool], None] | None = None,
    ) -> None: ...
    async def serve_forever(self, *, background: bool = False) -> None: ...

class ModbusSerialServer:
    context: ModbusServerContext | SimCore
    def __init__(
        self,
        context: ModbusServerContext | SimDevice | list[SimDevice],
        *,
        port: str = "",
        stopbits: int = 1,
        bytesize: int = 8,
        parity: str = "N",
        baudrate: int = 19200,
        ignore_missing_devices: bool = False,
    ) -> None: ...
    async def serve_forever(self, *, background: bool = False) -> None: ...
