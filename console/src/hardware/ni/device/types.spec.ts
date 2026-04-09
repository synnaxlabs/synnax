// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { propertiesZ, ZERO_PROPERTIES } from "@/hardware/ni/device/types";

describe("NI Device propertiesZ", () => {
  it("should parse ZERO_PROPERTIES", () => {
    expect(propertiesZ.safeParse(ZERO_PROPERTIES).success).toBe(true);
  });

  it("should parse device properties missing counterInput (pre-SY-3060)", () => {
    const oldProps = {
      identifier: "Dev1",
      analogInput: { portCount: 4, index: 0, channels: {} },
      analogOutput: { portCount: 2, stateIndex: 0, channels: {} },
      // counterInput missing — added in SY-3060
      digitalInputOutput: { portCount: 2, lineCounts: [8, 8] },
      digitalInput: { portCount: 2, lineCounts: [8, 8], index: 0, channels: {} },
      digitalOutput: { portCount: 2, lineCounts: [8, 8], stateIndex: 0, channels: {} },
    };
    const result = propertiesZ.safeParse(oldProps);
    expect(result.success).toBe(true);
  });

  it("should parse device with partially populated analogOutput", () => {
    const props = {
      identifier: "Dev1",
      analogInput: { portCount: 4, index: 0, channels: {} },
      analogOutput: { portCount: 2 }, // missing stateIndex, channels
      counterInput: { portCount: 0, index: 0, channels: {} },
      digitalInputOutput: { portCount: 0, lineCounts: [] },
      digitalInput: { portCount: 0, lineCounts: [], index: 0, channels: {} },
      digitalOutput: { portCount: 0, lineCounts: [], stateIndex: 0, channels: {} },
    };
    const result = propertiesZ.safeParse(props);
    expect(result.success).toBe(true);
  });

  it("should parse completely empty properties", () => {
    const result = propertiesZ.safeParse({});
    expect(result.success).toBe(true);
  });
});
