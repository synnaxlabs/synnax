// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { payload } from "@/payload";
import { DataType } from "@/telem";

describe("kindOf", () => {
  it.each([
    [DataType.FLOAT32, "float"],
    [DataType.FLOAT64, "float"],
    [DataType.INT8, "integer"],
    [DataType.INT64, "integer"],
    [DataType.UINT8, "unsigned"],
    [DataType.UINT64, "unsigned"],
    [DataType.BOOLEAN, "boolean"],
    [DataType.STRING, "string"],
    [DataType.TIMESTAMP, "timestamp"],
    [DataType.JSON, "json"],
    [DataType.UUID, "uuid"],
    [DataType.BYTES, "bytes"],
  ])("should map %s to %s", (dataType, kind) => {
    expect(payload.kindOf(dataType)).toBe(kind);
  });

  it("should not map an unknown data type", () => {
    expect(payload.kindOf(DataType.UNKNOWN)).toBeUndefined();
  });
});

describe("isNumeric", () => {
  it("should hold for the float, integer, and unsigned kinds only", () => {
    expect(payload.KINDS.filter(payload.isNumeric)).toEqual([
      "float",
      "integer",
      "unsigned",
    ]);
  });
});
