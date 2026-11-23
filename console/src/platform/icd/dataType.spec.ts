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

import { ICD } from "@/platform/icd";

describe("ICD data types", () => {
  describe("converts", () => {
    it.each([
      ["float", DataType.FLOAT64, true],
      ["integer", DataType.FLOAT32, true],
      ["unsigned", DataType.INT16, true],
      ["boolean", DataType.UINT8, true],
      ["boolean", DataType.FLOAT64, true],
      ["integer", DataType.UINT32, false],
      ["float", DataType.INT64, false],
      ["string", DataType.STRING, true],
      ["string", DataType.FLOAT64, false],
      ["timestamp", DataType.TIMESTAMP, true],
      ["timestamp", DataType.INT64, false],
    ] as const)("should report whether %s converts to %s", (kind, dt, expected) => {
      expect(ICD.converts(kind, dt.toString())).toBe(expected);
    });
  });

  describe("bindingDataType", () => {
    it("should keep the data type of a binding that names one", () => {
      expect(ICD.bindingDataType({ dataType: "float32" }, "float")).toBe("float32");
    });

    it("should use the default data type of the kind for a binding that names none", () => {
      expect(ICD.bindingDataType({ dataType: "" }, "unsigned")).toBe("uint64");
    });
  });
});
