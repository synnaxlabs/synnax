#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

class ModbusDeviceIdentification:
    VendorName: str
    ProductCode: str
    MajorMinorRevision: str
    VendorUrl: str
    ProductName: str
    ModelName: str
    def __init__(
        self,
        info: dict[int, str] | None = None,
        info_name: dict[str, str] | None = None,
    ) -> None: ...
