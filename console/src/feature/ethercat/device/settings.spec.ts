// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createSettingsDescription } from "@/feature/ethercat/testutil";

const { Settings } = EtherCAT.Device;
const { sameAddress } = Settings;

const MEASUREMENT = { index: 0x8040, subIndex: 1 };
const RANGE = { index: 0x8040, subIndex: 3 };
const EXCITATION = { index: 0x8040, subIndex: 0x1c };

const channel = createSettingsDescription().channels[0];
const fields = channel.settings;

describe("Settings", () => {
  describe("fill", () => {
    it("should give each address one value in field order, from the defaults", () => {
      expect(Settings.fill(fields, [])).toEqual([
        { ...MEASUREMENT, dataType: "uint16", value: 0, name: "" },
        { ...RANGE, dataType: "uint16", value: 0, name: "" },
        { ...EXCITATION, dataType: "float32", value: 0, name: "" },
      ]);
    });

    it("should start a field from the value that the card holds", () => {
      const held = fields.map((f) =>
        sameAddress(f, MEASUREMENT) ? { ...f, value: 6 } : f,
      );
      expect(Settings.fill(held, []).map(({ value }) => value)).toEqual([6, 0, 0]);
      expect(
        Settings.fill(held, [{ ...MEASUREMENT, value: 1 }]).map(({ value }) => value),
      ).toEqual([1, 0, 0]);
    });

    it("should drop a value at an address that no field names", () => {
      const settings = Settings.fill(fields, [
        { index: 0x8041, subIndex: 1, value: 6 },
      ]);
      expect(settings.map(({ value }) => value)).toEqual([0, 0, 0]);
    });
  });

  describe("findInput", () => {
    it("should find the input of each entry that reads it", () => {
      const description = createSettingsDescription();
      for (const entry of channel.entries)
        expect(Settings.findInput(description, entry)).toBe(description.channels[0]);
      expect(
        Settings.findInput(description, { index: 0x6040, subIndex: 2 }),
      ).toBeUndefined();
    });
  });

  describe("set", () => {
    it("should keep the values of other inputs", () => {
      const other = {
        index: 0x8041,
        subIndex: 1,
        dataType: "uint16",
        value: 6,
        name: "",
      };
      expect(Settings.set(fields, [other], RANGE, 1)).toContainEqual(other);
    });

    it("should give a shown field its default when its value is not an option", () => {
      const voltage = Settings.set(fields, [], RANGE, 4);
      const current = Settings.set(fields, voltage, MEASUREMENT, 1);
      expect(current.find((s) => Settings.sameAddress(s, RANGE))?.value).toBe(0);
    });

    it("should keep a value that the new options hold", () => {
      const voltage = Settings.set(fields, [], RANGE, 1);
      const current = Settings.set(fields, voltage, MEASUREMENT, 1);
      expect(current.find((s) => Settings.sameAddress(s, RANGE))?.value).toBe(1);
    });

    it("should give a hidden field its default when its value is outside its limits", () => {
      const bridge = Settings.set(fields, [], MEASUREMENT, 6);
      const high = Settings.set(fields, bridge, EXCITATION, 30);
      const voltage = Settings.set(fields, high, MEASUREMENT, 0);
      expect(voltage.find((s) => Settings.sameAddress(s, EXCITATION))?.value).toBe(0);
    });

    it("should keep a value of a hidden field that is inside its limits", () => {
      const bridge = Settings.set(fields, [], MEASUREMENT, 6);
      const low = Settings.set(fields, bridge, EXCITATION, 10);
      const voltage = Settings.set(fields, low, MEASUREMENT, 0);
      expect(voltage.find((s) => Settings.sameAddress(s, EXCITATION))?.value).toBe(10);
    });
  });

  describe("visibleFields", () => {
    it("should show the fields whose conditions hold", () => {
      const bridge = Settings.set(fields, [], MEASUREMENT, 6);
      expect(Settings.visibleFields(fields, bridge).map((f) => f.name)).toEqual([
        "Measurement",
        "Range",
        "Excitation voltage",
      ]);
      expect(Settings.visibleFields(fields, []).map((f) => f.name)).toEqual([
        "Measurement",
        "Range",
      ]);
    });
  });

  describe("check", () => {
    it("should report a broken requirement at the address it names", () => {
      const bridge = Settings.set(fields, [], MEASUREMENT, 6);
      expect(Settings.check(channel, bridge)).toEqual([
        { ...EXCITATION, message: "A bridge needs an excitation voltage above 0 V." },
      ]);
    });

    it("should report a value outside the limits of its field", () => {
      const bridge = Settings.set(fields, [], MEASUREMENT, 6);
      const high = Settings.set(fields, bridge, EXCITATION, 30);
      expect(Settings.check(channel, high)).toEqual([
        { ...EXCITATION, message: "Excitation voltage must be from 0 to 24 V." },
      ]);
    });

    it("should not check the limits of a hidden field", () => {
      const high = Settings.set(fields, [], EXCITATION, 30);
      expect(Settings.check(channel, high)).toEqual([]);
    });
  });
});
