// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { testPropertiesSchema } from "@/hardware/common/device/testutil";
import {
  slavePropertiesZ,
  ZERO_SLAVE_PROPERTIES,
} from "@/hardware/ethercat/device/types";

testPropertiesSchema("EtherCAT", slavePropertiesZ, ZERO_SLAVE_PROPERTIES, [
  [
    "scan-only properties (identifier and position only)",
    { identifier: "EL3004", position: 1 },
  ],
  [
    "properties missing PDOs",
    {
      identifier: "EL3004",
      serial: 12345,
      vendorId: 2,
      productCode: 0x0bbc3052,
      revision: 0,
      name: "EL3004",
      network: "eth0",
      position: 1,
    },
  ],
]);
