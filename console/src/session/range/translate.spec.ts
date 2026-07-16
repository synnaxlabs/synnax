// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { TimeRange, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Range } from "@/session/range";

const payload = (
  key: string,
  name: string,
  start: number,
  end: number,
): ranger.Payload => ({
  key,
  name,
  timeRange: new TimeRange(new TimeStamp(start), new TimeStamp(end)),
});

describe("range translate", () => {
  describe("fromClient", () => {
    it("should translate a single payload into a persisted static range", () => {
      const [result] = Range.fromClient(payload("r1", "Range 1", 0, 1000));
      expect(result).toEqual({
        variant: "static",
        key: "r1",
        name: "Range 1",
        timeRange: { start: 0, end: 1000 },
        persisted: true,
      });
    });

    it("should translate an array of payloads preserving order", () => {
      const result = Range.fromClient([
        payload("r1", "Range 1", 0, 1000),
        payload("r2", "Range 2", 500, 1500),
      ]);
      expect(result.map((r) => r.key)).toEqual(["r1", "r2"]);
      expect(result.every((r) => r.variant === "static" && r.persisted)).toBe(true);
    });

    it("should carry the numeric time range through", () => {
      const [result] = Range.fromClient(payload("r1", "Range 1", 42, 99));
      expect(result.variant).toBe("static");
      if (result.variant === "static")
        expect(result.timeRange).toEqual({ start: 42, end: 99 });
    });

    it("should return an empty array for an empty input array", () => {
      expect(Range.fromClient([])).toEqual([]);
    });
  });
});
