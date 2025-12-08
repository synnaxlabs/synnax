// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { nameZ, newZ } from "@/channel/payload";

describe("nameZ", () => {
  describe("valid names", () => {
    const validNames = [
      ["temperature", "lowercase letters"],
      ["Pressure", "capitalized word"],
      ["sensor1", "letters followed by digit"],
      ["sensor_temp", "letters with underscore"],
      ["temp123", "letters followed by digits"],
      ["Sensor_temp", "capitalized with underscore"],
      ["temp123_sensor_temp", "complex valid name"],
      ["_private", "underscore prefix"],
      ["__double", "double underscore prefix"],
      ["a", "single letter"],
      ["A", "single capital letter"],
      ["_", "single underscore"],
      ["_1", "underscore followed by digit"],
    ];
    validNames.forEach(([name, description]) => {
      it(`should accept ${name} (${description})`, () => {
        const result = nameZ.safeParse(name);
        expect(result.success).toBe(true);
      });
    });
  });

  describe("invalid names", () => {
    it("should reject empty string", () => {
      const result = nameZ.safeParse("");
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toContain("Name must not be empty");
    });

    it("should reject name starting with digit", () => {
      const result = nameZ.safeParse("1sensor");
      expect(result.success).toBe(false);
      // Regex validation covers both "cannot start with digit" and "invalid characters"
      expect(result.error?.issues[0].message).toContain(
        "can only contain letters, digits, and underscores",
      );
    });

    it("should reject name with spaces", () => {
      const result = nameZ.safeParse("my channel");
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toContain(
        "can only contain letters, digits, and underscores",
      );
    });

    it("should reject name with special characters", () => {
      const result = nameZ.safeParse("sensor!");
      expect(result.success).toBe(false);
    });

    it("should reject name with hyphens", () => {
      const result = nameZ.safeParse("sensor-temp");
      expect(result.success).toBe(false);
    });

    it("should reject name with dots", () => {
      const result = nameZ.safeParse("sensor.temp");
      expect(result.success).toBe(false);
    });

    it("should reject name with parentheses", () => {
      const result = nameZ.safeParse("sensor(1)");
      expect(result.success).toBe(false);
    });

    it("should reject name with brackets", () => {
      const result = nameZ.safeParse("sensor[0]");
      expect(result.success).toBe(false);
    });
  });
});

describe("newZ", () => {
  const validNewChannel = {
    name: "temperature_sensor",
    dataType: DataType.FLOAT32,
    virtual: true,
  };

  describe("name validation", () => {
    it("should accept valid channel names", () => {
      const result = newZ.safeParse(validNewChannel);
      expect(result.success).toBe(true);
    });

    it("should reject empty name", () => {
      const result = newZ.safeParse({ ...validNewChannel, name: "" });
      expect(result.success).toBe(false);
    });

    it("should reject name starting with digit", () => {
      const result = newZ.safeParse({ ...validNewChannel, name: "1sensor" });
      expect(result.success).toBe(false);
    });

    it("should reject name with spaces", () => {
      const result = newZ.safeParse({ ...validNewChannel, name: "my channel" });
      expect(result.success).toBe(false);
    });

    it("should reject name with special characters", () => {
      const result = newZ.safeParse({ ...validNewChannel, name: "sensor-temp" });
      expect(result.success).toBe(false);
    });

    it("should accept name with underscores", () => {
      const result = newZ.safeParse({
        ...validNewChannel,
        name: "sensor_temp_123",
      });
      expect(result.success).toBe(true);
    });

    it("should accept name starting with underscore", () => {
      const result = newZ.safeParse({ ...validNewChannel, name: "_private_sensor" });
      expect(result.success).toBe(true);
    });
  });
});
