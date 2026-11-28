// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

const fieldOf = (type: icd.JSONField["type"], extra: object = {}): icd.JSONField =>
  icd.jsonFieldZ.parse({
    name: "value",
    pointer: "/value",
    type,
    ...(type === "timestamp" ? { timeFormat: "unix_ms" } : {}),
    ...extra,
  });

const UUID = "4a0e9f6c-2b8e-4c3d-9f1a-6d2e8b7c5a10";

describe("ICD.parseDefault", () => {
  it.each([
    ["float", "2.5", 2.5],
    ["float", "-1e3", -1000],
    ["integer", "-12", "-12"],
    ["unsigned", "7", "7"],
    ["string", " idle ", " idle "],
    ["bytes", "AQI=", "AQI="],
    ["boolean", "True", true],
    ["boolean", "false", false],
    ["timestamp", "now", { type: "now" }],
    ["uuid", "Random", { type: "random" }],
    ["uuid", UUID, { type: "constant", value: UUID }],
    ["json", '{"a":1}', '{"a":1}'],
  ] as const)(
    "should give a %s field the default that %j spells",
    (type, text, value) => {
      expect(ICD.parseDefault(fieldOf(type), text).default).toEqual(value);
    },
  );

  it("should give a timestamp field a constant default from a date", () => {
    const field = ICD.parseDefault(fieldOf("timestamp"), "2026-01-02T03:04:05Z");
    expect(field.default).toEqual({
      type: "constant",
      value: new TimeStamp("2026-01-02T03:04:05Z"),
    });
  });

  it("should give an enum field the value of a label", () => {
    const field = fieldOf("integer", {
      enumValues: [
        { label: "idle", value: "0" },
        { label: "running", value: "1" },
      ],
    });
    expect(ICD.parseDefault(field, "running").default).toBe("1");
    expect(ICD.parseDefault(field, "0").default).toBe("0");
  });

  it("should keep a field as it is for empty text", () => {
    const field = fieldOf("float");
    expect(ICD.parseDefault(field, "")).toBe(field);
  });

  it.each([
    ["float", "warm", "Enter a number."],
    ["float", " ", "Enter a number."],
    ["integer", "1.5", "Enter a whole number."],
    ["unsigned", "-1", "Enter a whole number of 0 or more."],
    ["bytes", "not base64", "Enter base64."],
    ["boolean", "yes", "Enter true or false."],
    ["timestamp", "soon", "Enter now or a date."],
    ["uuid", "pump", "Enter random or a UUID."],
    ["json", "{", "Enter JSON."],
  ] as const)("should refuse a %s field the text %j", (type, text, message) => {
    expect(() => ICD.parseDefault(fieldOf(type), text)).toThrow(message);
  });
});
