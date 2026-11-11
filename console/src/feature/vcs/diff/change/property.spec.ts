// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Change } from "@/feature/vcs/diff/change";

const DEVICE = "3c7d1f2a-9b4e-4f6a-8c21-5e0d7b9a1f34";

const NAMES = new Map([
  ["channel:1048577", "pressure_1"],
  ["channel:1048578", "pressure_2"],
  ["channel:1048579", "valve_cmd"],
  [`device:${DEVICE}`, "NI 9237 strain module"],
  ["rack:65537", "cRIO 1"],
]);
const name: Change.Name = (id) => NAMES.get(resource.idToString(id));

const lines = (before: unknown, after: unknown, opts?: Change.DiffOptions) =>
  Change.diffProperties(before, after, { name, ...opts }).map(Change.format);

describe("Change", () => {
  describe("humanize", () => {
    it("should turn snake_case into sentence case", () => {
      expect(Change.humanize("data_saving")).toBe("Data saving");
    });

    it("should use the label of a field whose words alone do not read well", () => {
      expect(Change.humanize("voltage_excit_val")).toBe("Excitation (V)");
      expect(Change.humanize("sampleRate")).toBe("Sample rate (Hz)");
    });

    it("should turn camelCase into sentence case", () => {
      expect(Change.humanize("stateChannel")).toBe("State channel");
    });

    it("should keep acronyms upper case", () => {
      expect(Change.humanize("device_id")).toBe("Device ID");
      expect(Change.humanize("ai_voltage")).toBe("AI voltage");
    });
  });

  describe("diffProperties", () => {
    it("should report a changed field with its old and new values", () => {
      expect(lines({ sample_rate: 1000 }, { sample_rate: 2500 })).toEqual([
        "Sample rate (Hz): 1000 -> 2500",
      ]);
    });

    it("should report nothing for equal values", () => {
      expect(lines({ a: 1, b: null, c: [1, 2] }, { a: 1, b: null, c: [1, 2] })).toEqual(
        [],
      );
    });

    it("should leave the top-level config out of a path", () => {
      expect(
        lines({ config: { stream_rate: 25 } }, { config: { stream_rate: 50 } }),
      ).toEqual(["Stream rate (Hz): 25 -> 50"]);
    });

    it("should compare a config held as JSON text as an object", () => {
      expect(
        lines(
          { config: JSON.stringify({ sample_rate: 1000, data_saving: true }) },
          { config: JSON.stringify({ sample_rate: 2500, data_saving: true }) },
        ),
      ).toEqual(["Sample rate (Hz): 1000 -> 2500"]);
    });

    it("should show a channel key as the channel name", () => {
      expect(lines({ state_channel: 1048577 }, { state_channel: 1048578 })).toEqual([
        "State channel: pressure_1 -> pressure_2",
      ]);
    });

    it("should show an unknown channel key as the key", () => {
      expect(lines({ channel: 1048577 }, { channel: 999 })).toEqual([
        "Channel: pressure_1 -> 999",
      ]);
    });

    it("should show a list of channel keys as channel names", () => {
      expect(lines({ channels: [1048577] }, { channels: [1048577, 1048578] })).toEqual([
        "Channels: pressure_1 -> pressure_1, pressure_2",
      ]);
    });

    it("should link a channel key to the channel", () => {
      expect(
        Change.diffProperties({ channel: 1048577 }, { channel: 1048578 }, { name }),
      ).toEqual([
        {
          path: ["Channel"],
          kind: "changed",
          before: [
            { text: "pressure_1", resource: { type: "channel", key: "1048577" } },
          ],
          after: [
            { text: "pressure_2", resource: { type: "channel", key: "1048578" } },
          ],
        },
      ]);
    });

    it("should link each channel of a list and keep the separators plain", () => {
      const [property] = Change.diffProperties(
        { channels: [] },
        { channels: [1048577, 1048578] },
        { name },
      );
      expect(property.after).toEqual([
        { text: "pressure_1", resource: { type: "channel", key: "1048577" } },
        { text: ", " },
        { text: "pressure_2", resource: { type: "channel", key: "1048578" } },
      ]);
    });

    it("should show a device key as a link to the device", () => {
      const [property] = Change.diffProperties(
        { device: "" },
        { device: DEVICE },
        { name },
      );
      expect(property.after).toEqual([
        { text: "NI 9237 strain module", resource: { type: "device", key: DEVICE } },
      ]);
      expect(Change.format(property)).toBe("Device: Empty -> NI 9237 strain module");
    });

    it("should show a rack key as the rack name", () => {
      expect(lines({ rack: 65537 }, { rack: 65538 })).toEqual([
        "Rack: cRIO 1 -> 65538",
      ]);
    });

    it("should show the key of a device that does not exist as plain text", () => {
      const [property] = Change.diffProperties(
        { device: DEVICE },
        { device: "missing" },
        { name },
      );
      expect(property.after).toEqual([{ text: "missing" }]);
    });

    it("should not resolve a number in a field that holds no channel", () => {
      expect(lines({ port: 1048577 }, { port: 1048578 })).toEqual([
        "Port: 1048577 -> 1048578",
      ]);
    });

    it("should match keyed elements by key and name them", () => {
      const before = {
        channels: [
          { key: "a", port: 0, channel: 1048577, enabled: true },
          { key: "b", port: 1, channel: 1048578, enabled: true },
        ],
      };
      const after = {
        channels: [
          { key: "b", port: 1, channel: 1048578, enabled: false },
          { key: "a", port: 0, channel: 1048577, enabled: true },
        ],
      };
      expect(lines(before, after)).toEqual([
        "Channels › pressure_2 › Enabled: Yes -> No",
      ]);
    });

    it("should report an added and a removed element", () => {
      const before = { channels: [{ key: "a", port: 0, channel: 1048577 }] };
      const after = { channels: [{ key: "b", port: 3 }] };
      expect(lines(before, after)).toEqual([
        "Channels › pressure_1: removed",
        "Channels › Port 3: added",
      ]);
    });

    it("should match elements without keys by position", () => {
      expect(
        lines({ rules: [{ min: 0 }, { min: 5 }] }, { rules: [{ min: 0 }, { min: 6 }] }),
      ).toEqual(["Rules › Item 2 › Min: 5 -> 6"]);
    });

    it("should collapse a field that repeats its parent's name", () => {
      expect(
        lines({ label: { label: "Valve" } }, { label: { label: "Valve 2" } }),
      ).toEqual(["Label: Valve -> Valve 2"]);
    });

    it("should show a missing value as none", () => {
      expect(lines({}, { offset: 5 })).toEqual(["Offset: None -> 5"]);
    });

    it("should show booleans and empty text in words", () => {
      expect(
        lines({ auto_start: false, unit: "" }, { auto_start: true, unit: "V" }),
      ).toEqual(["Auto start: No -> Yes", "Unit: Empty -> V"]);
    });

    it("should leave out omitted fields", () => {
      expect(
        lines({ key: "a", name: "x" }, { key: "b", name: "y" }, { omit: ["key"] }),
      ).toEqual(["Name: x -> y"]);
    });
  });

  describe("listProperties", () => {
    it("should list every value of a created resource but keys", () => {
      expect(
        Change.listProperties(
          {
            key: "t1",
            name: "Read",
            config: { sample_rate: 1000, channels: [{ key: "a", channel: 1048577 }] },
          },
          "added",
          { name, omit: ["key"] },
        ).map(Change.format),
      ).toEqual([
        "Name: Read",
        "Sample rate (Hz): 1000",
        "Channels › pressure_1 › Channel: pressure_1",
      ]);
    });

    it("should put each value of a deleted resource in before", () => {
      expect(Change.listProperties({ rate: 5 }, "removed")).toEqual([
        { path: ["Rate"], kind: "removed", before: [{ text: "5" }] },
      ]);
    });
  });

  describe("references", () => {
    it("should collect the channels, devices, and racks, config text included", () => {
      expect(
        Change.references(
          {
            rack: 65537,
            config: JSON.stringify({
              channels: [{ channel: 1048577, device: DEVICE, port: 2 }],
            }),
          },
          { state_channel: 1048578, index: 1048579, port: 7, device: DEVICE },
        ),
      ).toEqual([
        { type: "rack", key: "65537" },
        { type: "channel", key: "1048577" },
        { type: "device", key: DEVICE },
        { type: "channel", key: "1048578" },
        { type: "channel", key: "1048579" },
      ]);
    });

    it("should skip an empty device key", () => {
      expect(Change.references({ device: "", channel: 0 })).toEqual([]);
    });
  });
});
