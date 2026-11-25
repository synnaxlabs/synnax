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
        expect(channel.nameZ.validate(name)).toBe(true);
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

describe("schema validation", () => {
  const validNewChannel: channel.New = {
    name: "temperature_sensor",
    dataType: DataType.FLOAT32,
    virtual: true,
  };

  describe("name validation", () => {
    it("should accept valid channel names", () => {
      expect(channel.payloadZ.validate(validNewChannel)).toBe(true);
    });

    it("should reject empty name", () => {
      expect(channel.payloadZ.validate({ ...validNewChannel, name: "" })).toBe(false);
    });

    it("should accept name with underscores", () => {
      const result = channel.payloadZ.safeParse({
        ...validNewChannel,
        name: "sensor_temp_123",
      });
      expect(result.success).toBe(true);
    });

    it("should accept name starting with underscore", () => {
      const result = channel.payloadZ.safeParse({
        ...validNewChannel,
        name: "_private_sensor",
      });
      expect(result.success).toBe(true);
    });
  });
});

describe("isValidName", () => {
  it.each([
    ["a dash", "sensor-temp"],
    ["a leading digit", "1sensor"],
    ["a space", "my channel"],
    ["a symbol", "sensor!"],
    ["a letter outside ASCII", "temp\u00e9rature"],
  ])("should allow a name with %s", (_, name) => {
    expect(channel.isValidName(name)).toBe(true);
  });

  it.each([
    ["an empty name", ""],
    ["spaces alone", "  "],
    ["a dot", "sensor.temp"],
    ["a tab", "sensor\ttemp"],
    ["a space other than U+0020", "sensor\u00a0temp"],
    ["a character that shows as nothing", "sensor\u200btemp"],
    ["a name that is not in NFC", "e\u0301"],
  ])("should refuse %s", (_, name) => {
    expect(channel.isValidName(name)).toBe(false);
  });
});

describe("escapeInvalidName", () => {
  it.each([
    ["a dash", "sensor-temp"],
    ["a leading digit", "1sensor"],
    ["a space", "my channel"],
    ["a symbol", "sensor!"],
  ])("should keep a name with %s", (_, name) => {
    expect(channel.escapeInvalidName(name)).toBe(name);
  });

  it.each([
    ["a dot", "sensor.temp", "sensor_temp"],
    ["a tab", "sensor\ttemp", "sensor_temp"],
    ["a space other than U+0020", "sensor\u00a0temp", "sensor_temp"],
    ["a character that shows as nothing", "sensor\u200btemp", "sensor_temp"],
    ["spaces alone", "  ", "__"],
  ])("should replace %s with an underscore", (_, name, escaped) => {
    expect(channel.escapeInvalidName(name)).toBe(escaped);
  });

  it("should put a name in NFC", () => {
    expect(channel.escapeInvalidName("e\u0301")).toBe("\u00e9");
  });

  it("should give a name that the rule allows", () => {
    expect(channel.isValidName(channel.escapeInvalidName("a.b\u200b c"))).toBe(true);
  });

  it("should allow an empty string by default", () => {
    expect(channel.escapeInvalidName("")).toBe("");
  });

  it("should change an empty string to an underscore when asked", () => {
    expect(channel.escapeInvalidName("", true)).toBe("_");
  });
});

describe("singleWhere", () => {
  it("should select a key by key", () => {
    expect(channel.singleWhere(12)).toEqual({ key: { eq: 12 } });
  });

  it("should select a name by name", () => {
    expect(channel.singleWhere("sensor")).toEqual({ name: { eq: "sensor" } });
  });

  it("should read a string with leading digits as a name", () => {
    expect(channel.singleWhere("12abc")).toEqual({ name: { eq: "12abc" } });
  });
});
