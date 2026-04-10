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

describe("SY-4047: NI propertiesZ rejects valid old data", () => {
  it("should reject ZERO_PROPERTIES (identifier fails .min(2))", () => {
    // This proves propertiesZ can't even parse its own zero state.
    const result = propertiesZ.safeParse(ZERO_PROPERTIES);
    expect(result.success).toBe(false);
  });

  it("should reject properties missing counterInput (pre-SY-3060 data)", () => {
    // A device created before v0.39 wouldn't have counterInput.
    const oldProps = {
      identifier: "Dev1",
      analogInput: { portCount: 4, index: 0, channels: {} },
      analogOutput: { portCount: 2, stateIndex: 0, channels: {} },
      // counterInput missing
      digitalInputOutput: { portCount: 2, lineCounts: [8, 8] },
      digitalInput: { portCount: 2, lineCounts: [8, 8], index: 0, channels: {} },
      digitalOutput: {
        portCount: 2,
        lineCounts: [8, 8],
        stateIndex: 0,
        channels: {},
      },
    };
    const result = propertiesZ.safeParse(oldProps);
    expect(result.success).toBe(false);
  });

  it("should reject partial analogOutput (enriched.json shallow merge)", () => {
    // enriched.json provides { portCount: 2 } which overwrites the full zero
    // object, losing stateIndex and channels.
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
    expect(result.success).toBe(false);
  });

  it("should reject empty properties (scan-only device)", () => {
    const result = propertiesZ.safeParse({});
    expect(result.success).toBe(false);
  });
});
