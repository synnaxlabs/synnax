// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Telem } from "@/telem";

describe("Telem.resolveDataTypeName", () => {
  it("should keep a numeric type lowercase", () => {
    expect(Telem.resolveDataTypeName(new DataType("float64"))).toBe("float64");
  });

  it("should capitalize a timestamp and a boolean", () => {
    expect(Telem.resolveDataTypeName(new DataType("timestamp"))).toBe("Timestamp");
    expect(Telem.resolveDataTypeName(new DataType("boolean"))).toBe("Boolean");
  });

  it("should capitalize a variable type", () => {
    expect(Telem.resolveDataTypeName(new DataType("string"))).toBe("String");
  });

  it("should write UUID and JSON in capitals", () => {
    expect(Telem.resolveDataTypeName(new DataType("uuid"))).toBe("UUID");
    expect(Telem.resolveDataTypeName(new DataType("json"))).toBe("JSON");
  });
});
