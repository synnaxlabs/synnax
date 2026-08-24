// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { staticWriteFieldZ } from "@/http/types.gen";

const FIELD = { key: "field-1", pointer: "/setpoint", type: "static" } as const;

describe("staticWriteFieldZ", () => {
  const CASES: [string, unknown][] = [
    ["string", "open"],
    ["number", 12.5],
    ["boolean", true],
    ["object", { unit: "psi" }],
    ["array", [1, 2, 3]],
  ];
  it.each(CASES)("should accept a %s value", (_, value) => {
    expect(staticWriteFieldZ.parse({ ...FIELD, value }).value).toEqual(value);
  });

  it("should accept an explicit null value", () => {
    expect(staticWriteFieldZ.parse({ ...FIELD, value: null }).value).toBeNull();
  });

  it("should reject a field with no value", () => {
    expect(() => staticWriteFieldZ.parse(FIELD)).toThrow();
  });
});
