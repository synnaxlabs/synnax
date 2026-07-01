// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { OPC } from "@/feature/opc";
import * as v0 from "@/feature/opc/device/types/v0";
import { testPropertiesSchema } from "@/platform/device/testutil";

// OPC uses versioned schemas — empty `{}` is not valid for either version.
testPropertiesSchema("OPC UA", OPC.Device.propertiesZ, OPC.Device.ZERO_PROPERTIES, [], {
  testEmpty: false,
});

describe("OPC UA propertiesZ v0 migration", () => {
  it("should migrate v0 properties through the union", () => {
    const result = OPC.Device.propertiesZ.safeParse(v0.ZERO_PROPERTIES);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.version).toBe("1.0.0");
      expect(result.data.read.indexes).toEqual([]);
    }
  });

  it("should migrate v0 read.index into read.indexes", () => {
    const v0Props = {
      ...v0.ZERO_PROPERTIES,
      read: { index: 42, channels: { "ns=2;s=Tag1": 100 } },
    };
    const result = OPC.Device.propertiesZ.safeParse(v0Props);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.version).toBe("1.0.0");
      expect(result.data.read.indexes).toEqual([42]);
      expect(result.data.read.channels).toEqual({ "ns=2;s=Tag1": 100 });
    }
  });
});
