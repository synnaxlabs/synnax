// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { labjack } from "@/labjack";
import { vectors } from "@/testutil/vectors";

describe("whereZ", () => {
  const { cases } = vectors.read(
    "labjackDevices",
    labjack.device.deviceZ,
    labjack.device.whereZ,
  );
  it.each(cases)("should parse the where of $name", ({ where }) => {
    expect(labjack.device.whereZ.safeParse(where).success).toBe(true);
  });

  it("should reject a where on a field the filter does not have", () => {
    expect(labjack.device.whereZ.safeParse({ connection: { eq: "x" } }).success).toBe(
      false,
    );
  });
});
