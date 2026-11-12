// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { color, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { formatsOf, formatZ, textOf } from "@/log/aether/format";

describe("log/aether/format", () => {
  describe("textOf", () => {
    it("should draw a value with no format as it arrives", () => {
      expect(textOf("3.14159", undefined)).toBe("3.14159");
    });

    it("should draw a number at its precision", () => {
      const format = formatZ.parse({ precision: 2, dataType: "float64" });
      expect(textOf("3.14159", format)).toBe("3.14");
    });

    it("should draw a number as it arrives with no precision or notation", () => {
      expect(textOf("3.14159", formatZ.parse({ dataType: "float64" }))).toBe("3.14159");
    });

    it("should draw a number as it arrives before its data type loads", () => {
      expect(textOf("3.14159", formatZ.parse({ precision: 2 }))).toBe("3.14159");
    });

    it("should draw a value that is not a number as it arrives", () => {
      const format = formatZ.parse({ precision: 2, dataType: "float64" });
      expect(textOf("n/a", format)).toBe("n/a");
    });

    it("should draw a timestamp in its format", () => {
      const time = TimeStamp.seconds(86_400);
      const format = formatZ.parse({
        dataType: "timestamp",
        timestampConfig: { format: "dateTime", timeZone: "UTC" },
      });
      expect(textOf(time.valueOf().toString(), format)).toBe(
        time.toString("dateTime", "UTC"),
      );
    });

    it("should draw a timestamp that does not parse as it arrives", () => {
      expect(textOf("soon", formatZ.parse({ dataType: "timestamp" }))).toBe("soon");
    });
  });

  describe("formatsOf", () => {
    it("should take the settings of each channel entry", () => {
      const channels = [
        log.channelEntryZ.parse({ channel: 1, color: color.ZERO, precision: 3 }),
      ];
      expect(formatsOf(channels, { "1": "float32" })["1"]).toMatchObject({
        precision: 3,
        dataType: "float32",
      });
    });

    it("should give the default format to a channel with no entry", () => {
      expect(formatsOf([], { "2": "timestamp" })["2"]).toEqual({
        ...formatZ.parse({}),
        dataType: "timestamp",
      });
    });
  });
});
