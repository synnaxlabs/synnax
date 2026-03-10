// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { SCAN_SCHEMAS } from "@/hardware/ethercat/task/types";

describe("EtherCAT Scan Task Types", () => {
  it("should parse null scan config as empty object", () => {
    const result = SCAN_SCHEMAS.configSchema.safeParse(null);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });

  it("should parse undefined scan config as empty object", () => {
    const result = SCAN_SCHEMAS.configSchema.safeParse(undefined);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });
});
