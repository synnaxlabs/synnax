// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { json } from "@/json";
import { DataType } from "@/telem";

const field = {
  pointer: "/ts",
  dataType: "timestamp",
  type: "number",
  timeFormat: "unix_ms",
  enum: [],
  count: 2,
  mismatched: 0,
};

describe("shapeZ", () => {
  it("should parse the index and the data type of each field", () => {
    const shape = json.shapeZ.parse({
      sources: ["tank-1"],
      fields: [field],
      index: 0,
      count: 2,
    });
    expect(shape.index).toEqual(0);
    expect(shape.fields[0].dataType.equals(DataType.TIMESTAMP)).toBe(true);
    expect(shape.fields[0].timeFormat).toEqual("unix_ms");
  });

  it("should read a negative index as no index", () => {
    const shape = json.shapeZ.parse({
      sources: [],
      fields: [field],
      index: -1,
      count: 2,
    });
    expect(shape.index).toBeUndefined();
  });

  it("should reject a field with a null type", () => {
    expect(() => json.inferredFieldZ.parse({ ...field, type: "null" })).toThrow();
  });
});
