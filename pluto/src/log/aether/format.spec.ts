// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { format } from "@/log/aether";

describe("log/aether/format", () => {
  describe("textOf", () => {
    it("should draw a value with no format as it arrives", () => {
      expect(format.textOf("3.14159", undefined)).toBe("3.14159");
    });

    it("should draw a number at its precision", () => {
      const drawn = format.formatZ.parse({ precision: 2, dataType: "float64" });
      expect(format.textOf("3.14159", drawn)).toBe("3.14");
    });

    it("should draw a number as it arrives with no precision or notation", () => {
      expect(
        format.textOf("3.14159", format.formatZ.parse({ dataType: "float64" })),
      ).toBe("3.14159");
    });

    it("should draw a number as it arrives before its data type loads", () => {
      expect(format.textOf("3.14159", format.formatZ.parse({ precision: 2 }))).toBe(
        "3.14159",
      );
    });

    it("should draw a value that is not a number as it arrives", () => {
      const drawn = format.formatZ.parse({ precision: 2, dataType: "float64" });
      expect(format.textOf("n/a", drawn)).toBe("n/a");
    });

    it("should draw a timestamp in its format", () => {
      const time = TimeStamp.seconds(86_400);
      const drawn = format.formatZ.parse({
        dataType: "timestamp",
        timestampConfig: { format: "dateTime", timeZone: "UTC" },
      });
      expect(format.textOf(time.valueOf().toString(), drawn)).toBe(
        time.toString("dateTime", "UTC"),
      );
    });

    it("should draw a timestamp that does not parse as it arrives", () => {
      expect(
        format.textOf("soon", format.formatZ.parse({ dataType: "timestamp" })),
      ).toBe("soon");
    });
  });

  describe("formatsOf", () => {
    it("should take the settings of each channel entry", () => {
      const channels = [log.channelEntryZ.parse({ channel: 1, precision: 3 })];
      expect(format.formatsOf(channels, { "1": "float32" })["1"]).toMatchObject({
        precision: 3,
        dataType: "float32",
      });
    });

    it("should give the default format to a channel with no entry", () => {
      expect(format.formatsOf([], { "2": "timestamp" })["2"]).toEqual({
        ...format.formatZ.parse({}),
        dataType: "timestamp",
      });
    });
  });
});
