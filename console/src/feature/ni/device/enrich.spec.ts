// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";

describe("enrich", () => {
  it("should leave properties untouched for an unknown model", () => {
    const enriched = NI.Device.enrich("NOT-A-MODEL", NI.Device.ZERO_PROPERTIES);
    expect(enriched).toEqual(NI.Device.ZERO_PROPERTIES);
  });

  it("should let existing device properties win over the estimated pinout", () => {
    const properties: NI.Device.Properties = {
      ...NI.Device.ZERO_PROPERTIES,
      identifier: "dev1",
      analogInput: { portCount: 4, index: 12, channels: { "0": 34 } },
    };
    const enriched = NI.Device.enrich("TS-15100", properties);
    expect(enriched.analogInput.portCount).toBe(4);
    expect(enriched.analogInput.index).toBe(12);
    expect(enriched.analogInput.channels).toEqual({ "0": 34 });
    expect(enriched.identifier).toBe("dev1");
  });

  it("should not mutate the input properties", () => {
    const properties: NI.Device.Properties = {
      ...NI.Device.ZERO_PROPERTIES,
      analogInput: { portCount: 0, index: 0, channels: {} },
    };
    NI.Device.enrich("TS-15100", properties);
    expect(properties.analogInput.portCount).toBe(0);
  });
});
