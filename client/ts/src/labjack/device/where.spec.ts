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
import { matches, serverComputed } from "@/labjack/device/where.gen";
import { vectors } from "@/testutil/vectors";

// No client namespace serves LabJack devices yet, so the cases run on the generated
// matcher alone.
describe("Where", () => {
  const { records, cases, whereZ } = vectors.read(
    "labjackDevices",
    labjack.device.deviceZ,
    labjack.device.whereZ,
  );
  it.each(cases)("should match $name", (c) => {
    const where = whereZ.parse(c.where);
    expect(serverComputed(where)).toEqual(c.serverComputed ?? false);
    if (c.serverComputed) return;
    const matched = records
      .filter(([, d]) => matches(d, where))
      .map(([alias]) => alias);
    expect(matched.sort()).toEqual([...c.expected].sort());
  });
});
