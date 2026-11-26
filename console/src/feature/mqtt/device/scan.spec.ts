// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { MQTT } from "@/feature/mqtt";

describe("MQTT Scan Task", () => {
  it("should accept a null and an absent config", () => {
    expect(z.validate(MQTT.Device.SCAN_SCHEMAS.config, null)).toBe(true);
    expect(z.validate(MQTT.Device.SCAN_SCHEMAS.config, undefined)).toBe(true);
  });

  it("should accept the null status data of a reply with no data", () => {
    expect(z.validate(MQTT.Device.SCAN_SCHEMAS.statusData, null)).toBe(true);
  });

  it("should keep the status data of a reply as the scan task sent it", () => {
    // A sample reply with no topics also holds every field of a browse reply.
    const shape = { sources: ["plant/temp"], fields: [{ pointer: "/v" }], count: 1 };
    const data = { topics: [], shapes: [shape], truncated: false };
    expect(MQTT.Device.SCAN_SCHEMAS.statusData.parse(data)).toEqual(data);
  });
});
