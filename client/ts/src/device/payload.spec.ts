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

import { deviceZ, newZ } from "@/device/types.gen";

const VALID_DEVICE = {
  key: "dev-1",
  rack: 1,
  name: "Test Device",
  make: "ni",
  model: "pxi-6281",
  location: "Lab1",
  properties: { rate: 10 },
  status: undefined,
};

describe("deviceZ", () => {
  describe("default schemas", () => {
    it("should accept a valid device", () => {
      const result = deviceZ().safeParse(VALID_DEVICE);
      expect(result.success).toBe(true);
    });

    it("should reject a device with an empty name", () => {
      const result = deviceZ().safeParse({ ...VALID_DEVICE, name: "" });
      expect(result.success).toBe(false);
    });

    it("should reject a device with an empty make", () => {
      const result = deviceZ().safeParse({ ...VALID_DEVICE, make: "" });
      expect(result.success).toBe(false);
    });

    it("should reject a device with an empty model", () => {
      const result = deviceZ().safeParse({ ...VALID_DEVICE, model: "" });
      expect(result.success).toBe(false);
    });

    it("should accept properties as an object", () => {
      const result = deviceZ().parse(VALID_DEVICE);
      expect(result.properties).toEqual({ rate: 10 });
    });
  });

  describe("custom make schema", () => {
    const makeZ = z.enum(["ni", "labjack"]);

    it("should accept a valid make value", () => {
      const result = deviceZ({ make: makeZ }).safeParse(VALID_DEVICE);
      expect(result.success).toBe(true);
    });

    it("should reject an invalid make value", () => {
      const result = deviceZ({ make: makeZ }).safeParse({
        ...VALID_DEVICE,
        make: "unknown",
      });
      expect(result.success).toBe(false);
    });
  });

  describe("custom model schema", () => {
    const modelZ = z.literal("pxi-6281");

    it("should accept a matching model", () => {
      const result = deviceZ({ model: modelZ }).safeParse(VALID_DEVICE);
      expect(result.success).toBe(true);
    });

    it("should reject a non-matching model", () => {
      const result = deviceZ({ model: modelZ }).safeParse({
        ...VALID_DEVICE,
        model: "t7",
      });
      expect(result.success).toBe(false);
    });
  });

  describe("custom properties schema", () => {
    const propertiesZ = z.object({ rate: z.number(), channel: z.string() });

    it("should validate properties against the custom schema", () => {
      const result = deviceZ({ properties: propertiesZ }).safeParse({
        ...VALID_DEVICE,
        properties: { rate: 10, channel: "ai0" },
      });
      expect(result.success).toBe(true);
    });

    it("should reject properties that don't match the custom schema", () => {
      expect(() => deviceZ({ properties: propertiesZ }).parse(VALID_DEVICE)).toThrow();
    });
  });
});

describe("newZ", () => {
  it("should encode properties to a JSON string", () => {
    const result = newZ().parse(VALID_DEVICE);
    expect(result.properties).toEqual(VALID_DEVICE.properties);
  });

  it("should still validate make and model with custom schemas", () => {
    const makeZ = z.literal("ni");
    const result = newZ({ make: makeZ }).safeParse(VALID_DEVICE);
    expect(result.success).toBe(true);
    const bad = newZ({ make: makeZ }).safeParse({ ...VALID_DEVICE, make: "opc" });
    expect(bad.success).toBe(false);
  });
});
