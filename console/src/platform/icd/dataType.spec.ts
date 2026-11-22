// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { DataType, uuid } from "@synnaxlabs/x";
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
    const fieldOf = (type: "float" | "integer" | "unsigned", scale = 1, offset = 0) =>
      icd.jsonFieldZ.parse({
        key: uuid.create(),
        name: "v",
        pointer: "/v",
        type,
        scale,
        offset,
      });

    it("should keep the data type of a binding that names one", () => {
      expect(ICD.bindingDataType({ dataType: "float32" }, fieldOf("float"))).toBe(
        "float32",
      );
    });

    it.each([
      ["an unscaled unsigned field", fieldOf("unsigned"), "uint64"],
      ["an integer field with a scale", fieldOf("integer", 0.5), "float64"],
      ["an unsigned field with an offset", fieldOf("unsigned", 1, 4), "float64"],
    ] as const)(
      "should give a binding that names no data type the default of %s",
      (_, field, expected) => {
        expect(ICD.bindingDataType({ dataType: "" }, field)).toBe(expected);
      },
    );
  });
});
