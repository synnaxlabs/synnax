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

describe("hardwareZ", () => {
  it.each([-1, Infinity, NaN])("should reject an analog input rate of %s", (rate) => {
    expect(() => ni.hardwareZ.parse({ aiMinRate: rate })).toThrow();
  });

  it("should accept an analog input rate of 0", () => {
    expect(ni.hardwareZ.parse({ aiMinRate: 0 }).aiMinRate).toBe(0);
  });
});
