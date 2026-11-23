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

  it("should parse the status data of a browse reply", () => {
    const data = {
      topics: [
        {
          topic: "plant/temp",
          payload: "21.5",
          fields: [{ pointer: "", type: "number", sample: "21.5" }],
          retained: true,
        },
      ],
      truncated: false,
    };
    expect(MQTT.Device.SCAN_SCHEMAS.statusData.parse(data)).toEqual(data);
  });

  it("should parse the edge nodes of a Sparkplug B browse reply", () => {
    const data = {
      nodes: [{ group: "plant", edgeNode: "line_1", devices: ["ovenA"] }],
      tags: [],
    };
    expect(MQTT.Device.SCAN_SCHEMAS.statusData.parse(data)).toEqual(data);
  });

  it("should parse the tags of a Sparkplug B browse reply", () => {
    const tag = {
      device: "",
      name: "Node Control/Rebirth",
      dataType: "boolean",
      value: "false",
      supported: true,
    };
    const data = { nodes: [], tags: [tag] };
    expect(MQTT.Device.SCAN_SCHEMAS.statusData.parse(data)).toEqual(data);
  });

  it("should accept the status data of a sample reply", () => {
    const data = {
      topics: [
        {
          topic: "plant/temp",
          messages: [{ time: 0, format: "json", payload: "21.5", retained: false }],
        },
      ],
      shapes: [],
      truncated: false,
    };
    expect(z.validate(MQTT.Device.SCAN_SCHEMAS.statusData, data)).toBe(true);
  });

  it("should keep the shapes of a sample reply with no topics", () => {
    const shape = { sources: ["plant/temp"], fields: [], index: -1, count: 1 };
    const data = { topics: [], shapes: [shape], truncated: false };
    expect(MQTT.Device.SCAN_SCHEMAS.statusData.parse(data)).toEqual({
      ...data,
      shapes: [{ ...shape, index: undefined }],
    });
  });

  it("should reject a browse reply with no truncated flag", () => {
    expect(z.validate(MQTT.Device.SCAN_SCHEMAS.statusData, { topics: [] })).toBe(false);
  });
});
