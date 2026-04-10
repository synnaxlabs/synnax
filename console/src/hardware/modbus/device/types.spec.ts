// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { testPropertiesSchema } from "@/hardware/common/device/testutil";
import { SCHEMAS, ZERO_PROPERTIES } from "@/hardware/modbus/device/types";

testPropertiesSchema("Modbus", SCHEMAS.properties, ZERO_PROPERTIES, [
  [
    "properties with only connection config",
    { connection: { host: "192.168.1.10", port: 502 } },
  ],
]);
