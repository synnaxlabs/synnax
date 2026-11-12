// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";

const issuesOf = (field: object) =>
  Task.Payload.deployWriteFieldZ
    .safeParse({ key: "f", pointer: "/f", ...field })
    .error?.issues.map(({ message, path }) => ({ message, path: path.join(".") })) ??
  [];

describe("deployWriteFieldZ", () => {
  it.each<[string, object]>([
    ["float", { type: "float", value: -0.5 }],
    ["integer", { type: "integer", value: "-9223372036854775808" }],
    ["unsigned", { type: "unsigned", value: "18446744073709551615" }],
    ["string", { type: "string", value: "" }],
    ["boolean", { type: "boolean", value: true }],
    ["json", { type: "json", value: '{"a": [1, null]}' }],
    ["json with duplicate names", { type: "json", value: '{"a": 1, "a": 2}' }],
    ["bytes", { type: "bytes", value: "AQI=" }],
    ["uuid", { type: "uuid" }],
    ["timestamp", { type: "timestamp", timeFormat: "unix_ms" }],
  ])("should accept a valid %s field", (_, field) => {
    expect(issuesOf(field)).toEqual([]);
  });

  it.each<[string, object, string]>([
    [
      "a non-finite float",
      { type: "float", value: "Infinity" },
      "Value must be finite",
    ],
    ["json that is not JSON", { type: "json", value: "{" }, "Value is not JSON"],
    ["empty json", { type: "json", value: "" }, "Value is not JSON"],
    [
      "json with a number past float64 range",
      { type: "json", value: '{"a": 1e400}' },
      "Value is not JSON",
    ],
    [
      "json with a lone surrogate",
      { type: "json", value: '{"\\ud800": 1}' },
      "Value is not JSON",
    ],
    [
      "bytes that are not base64",
      { type: "bytes", value: "AQ*" },
      "Value is not base64",
    ],
    ["unpadded base64", { type: "bytes", value: "AQI" }, "Value is not base64"],
    [
      "base64 with nonzero trailing bits",
      { type: "bytes", value: "AR==" },
      "Value is not base64",
    ],
  ])("should reject %s", (_, field, message) => {
    expect(issuesOf(field)).toEqual([{ message, path: "value" }]);
  });

  it("should reject an empty pointer", () => {
    expect(issuesOf({ type: "uuid", pointer: "" })).toEqual([
      { message: "Pointer cannot be empty", path: "pointer" },
    ]);
  });

  it("should reject a pointer that is not a JSON pointer", () => {
    expect(issuesOf({ type: "uuid", pointer: "f" })).toEqual([
      { message: "must be a valid JSON pointer (RFC 6901)", path: "pointer" },
    ]);
  });

  it("should reject an integer out of range", () => {
    expect(issuesOf({ type: "integer", value: "9223372036854775808" })).toHaveLength(1);
  });
});
