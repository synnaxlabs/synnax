// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

const base = { key: "f", name: "f", pointer: "/f", scale: 1, offset: 0, units: "" };

describe("numberOptionsOf", () => {
  it("should keep the options of a field of the same kind", () => {
    const field: icd.JSONField = {
      ...base,
      type: "integer",
      enumValues: [{ label: "ON", value: "1" }],
      default: "-1",
    };
    expect(ICD.numberOptionsOf(field, "integer")).toEqual({
      type: "integer",
      enumValues: [{ label: "ON", value: "1" }],
      default: "-1",
    });
  });

  it("should drop a value that parses but does not fit the same kind", () => {
    const field: icd.JSONField = {
      ...base,
      type: "unsigned",
      enumValues: [
        { label: "ON", value: "1" },
        { label: "NEG", value: "-1" },
        { label: "BIG", value: "18446744073709551616" },
        { label: "PARTIAL", value: "" },
      ],
      default: "-2",
    };
    expect(ICD.numberOptionsOf(field, "unsigned")).toEqual({
      type: "unsigned",
      enumValues: [
        { label: "ON", value: "1" },
        { label: "PARTIAL", value: "" },
      ],
      default: undefined,
    });
  });

  it("should carry the entries and default that a new number kind holds", () => {
    const field: icd.JSONField = {
      ...base,
      type: "float",
      enumValues: [
        { label: "ON", value: 1 },
        { label: "HALF", value: 0.5 },
        { label: "NEG", value: -1 },
      ],
      default: 2,
    };
    expect(ICD.numberOptionsOf(field, "unsigned")).toEqual({
      type: "unsigned",
      enumValues: [{ label: "ON", value: "1" }],
      default: "2",
    });
  });

  it("should drop an entry and a default outside the range of the new kind", () => {
    const field: icd.JSONField = {
      ...base,
      type: "float",
      enumValues: [
        { label: "HIGH", value: 2 ** 62 },
        { label: "OVER", value: 2 ** 63 },
      ],
      default: -(2 ** 64),
    };
    expect(ICD.numberOptionsOf(field, "integer")).toEqual({
      type: "integer",
      enumValues: [{ label: "HIGH", value: "4611686018427387904" }],
      default: undefined,
    });
  });

  it("should drop an integer entry that a float does not hold exactly", () => {
    const field: icd.JSONField = {
      ...base,
      type: "integer",
      enumValues: [
        { label: "MAX", value: "9223372036854775807" },
        { label: "ONE", value: "1" },
        { label: "PARTIAL", value: "" },
        { label: "HUGE", value: "9".repeat(400) },
      ],
      default: "9007199254740993",
    };
    expect(ICD.numberOptionsOf(field, "float")).toEqual({
      type: "float",
      enumValues: [{ label: "ONE", value: 1 }],
      default: undefined,
    });
  });

  it("should drop a NaN default on an integer kind", () => {
    const field: icd.JSONField = {
      ...base,
      type: "float",
      enumValues: [],
      default: NaN,
    };
    expect(ICD.numberOptionsOf(field, "integer")).toEqual({
      type: "integer",
      enumValues: [],
      default: undefined,
    });
  });

  it("should start a field that holds no number with no options", () => {
    const field: icd.JSONField = {
      ...base,
      type: "string",
      enumValues: [{ label: "ON", value: "on" }],
    };
    expect(ICD.numberOptionsOf(field, "float")).toEqual({
      type: "float",
      enumValues: [],
    });
  });
});
