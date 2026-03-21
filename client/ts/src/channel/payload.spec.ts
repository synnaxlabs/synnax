// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";

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
        const result = channel.nameZ.safeParse(name);
        expect(result.success).toBe(true);
      });
    });
  });

  describe("invalid names", () => {
    it("should reject empty string", () => {
      const result = channel.nameZ.safeParse("");
      expect(result.success).toBe(false);
      expect(result.error?.issues[0].message).toContain("Name is required");
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
      const result = channel.newZ.safeParse(validNewChannel);
      expect(result.success).toBe(true);
    });

    it("should reject empty name", () => {
      const result = channel.newZ.safeParse({ ...validNewChannel, name: "" });
      expect(result.success).toBe(false);
    });

    it("should accept name with underscores", () => {
      const result = channel.newZ.safeParse({
        ...validNewChannel,
        name: "sensor_temp_123",
      });
      expect(result.success).toBe(true);
    });

    it("should accept name starting with underscore", () => {
      const result = channel.newZ.safeParse({
        ...validNewChannel,
        name: "_private_sensor",
      });
      expect(result.success).toBe(true);
    });
  });
});
describe("escapeInvalidName", () => {
  it("should escape invalid name", () => {
    const result = channel.escapeInvalidName("sensor-temp");
    expect(result).toBe("sensor_temp");
  });
  it("should escape name starting with digit", () => {
    const result = channel.escapeInvalidName("1sensor");
    expect(result).toBe("_1sensor");
  });
  it("should escape name with spaces", () => {
    const result = channel.escapeInvalidName("my channel");
    expect(result).toBe("my_channel");
  });
  it("should escape name with special characters", () => {
    const result = channel.escapeInvalidName("sensor!");
    expect(result).toBe("sensor_");
  });
  it("should escape name with hyphens", () => {
    const result = channel.escapeInvalidName("sensor-temp");
    expect(result).toBe("sensor_temp");
  });
  it("should escape name with dots", () => {
    const result = channel.escapeInvalidName("sensor.temp");
    expect(result).toBe("sensor_temp");
  });
  it("should allow an empty string by default", () => {
    const result = channel.escapeInvalidName("");
    expect(result).toBe("");
  });
  it("should change empty string to underscore when changeEmptyToUnderscore is true", () => {
    const result = channel.escapeInvalidName("", true);
    expect(result).toBe("_");
  });
});
