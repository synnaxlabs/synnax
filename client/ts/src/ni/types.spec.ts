// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ni } from "@/ni";

const CHANNEL = { type: "ao_func_gen", key: "gen", port: 0 } as const;

describe("aoFuncGenChannelZ", () => {
  it("should accept a positive frequency", () => {
    expect(ni.aoFuncGenChannelZ.parse({ ...CHANNEL, frequency: 12.5 }).frequency).toBe(
      12.5,
    );
  });

  it.each([0, -5])("should reject a frequency of %d", (frequency) => {
    expect(ni.aoFuncGenChannelZ.safeParse({ ...CHANNEL, frequency }).success).toBe(
      false,
    );
  });

  it("should reject a missing frequency", () => {
    expect(ni.aoFuncGenChannelZ.safeParse(CHANNEL).success).toBe(false);
  });
});
