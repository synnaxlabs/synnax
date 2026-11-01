#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from ..pdu import ModbusPDU

class ModbusTcpClient:
    def __init__(
        self,
        host: str,
        *,
        port: int = 502,
        name: str = "comm",
        source_address: tuple[str, int] | None = None,
        reconnect_delay: float = 0.1,
        reconnect_delay_max: float = 300,
        timeout: float = 3,
        retries: int = 3,
    ) -> None: ...
    def connect(self) -> bool: ...
    def close(self) -> None: ...
    def read_coils(
        self,
        address: int,
        *,
        count: int = 1,
        device_id: int = 1,
        no_response_expected: bool = False,
    ) -> ModbusPDU: ...
    def read_holding_registers(
        self,
        address: int,
        *,
        count: int = 1,
        device_id: int = 1,
        no_response_expected: bool = False,
    ) -> ModbusPDU: ...

class ModbusSerialClient:
    def __init__(
        self,
        port: str,
        *,
        baudrate: int = 19200,
        bytesize: int = 8,
        parity: str = "N",
        stopbits: int = 1,
        timeout: float = 3,
        retries: int = 3,
    ) -> None: ...
    def connect(self) -> bool: ...
    def close(self) -> None: ...
    def read_holding_registers(
        self,
        address: int,
        *,
        count: int = 1,
        device_id: int = 1,
        no_response_expected: bool = False,
    ) -> ModbusPDU: ...
