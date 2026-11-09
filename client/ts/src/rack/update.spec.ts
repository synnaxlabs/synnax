// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


import { describe, expect, it } from "vitest";

import { rack } from "@/rack";

describe("Update", () => {
  it("should take null on an optional field to clear it", () => {
    const update: rack.Update = { version: null, group: null };
    expect(rack.updateZ.parse(update)).toEqual(update);
  });

  it("should reject null on a required field", () => {
    // @ts-expect-error - only an optional field takes null.
    const update: rack.Update = { name: null };
    expect(rack.updateZ.safeParse(update).success).toBe(false);
  });

  it("should take operators on a list", () => {
    const update: rack.Update = { integrations: { insert: ["labjack"] } };
    expect(rack.updateZ.parse(update)).toEqual(update);
  });

  it("should reject a plain value on a list", () => {
    // @ts-expect-error - a list takes operators.
    const update: rack.Update = { integrations: ["labjack"] };
    expect(rack.updateZ.safeParse(update).success).toBe(false);
  });
});
