// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { DataType, type payload, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

const fieldOf = (type: payload.Kind, scale = 1, offset = 0) =>
  icd.jsonFieldZ.parse({
    key: uuid.create(),
    name: "v",
    pointer: "/v",
    type,
    scale,
    offset,
  });

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
      expect(ICD.converts({ type: kind, scale: 1, offset: 0 }, dt.toString())).toBe(
        expected,
      );
    });

    it.each([
      ["an integer field with a scale", fieldOf("integer", 0.5), DataType.INT64, false],
      [
        "an integer field with a scale",
        fieldOf("integer", 0.5),
        DataType.FLOAT32,
        true,
      ],
      [
        "an unsigned field with an offset",
        fieldOf("unsigned", 1, 4),
        DataType.UINT64,
        false,
      ],
      [
        "an unsigned field with an offset",
        fieldOf("unsigned", 1, 4),
        DataType.FLOAT64,
        true,
      ],
    ] as const)("should report whether %s converts to %s", (_, field, dt, expected) => {
      expect(ICD.converts(field, dt.toString())).toBe(expected);
    });
  });

  describe("encodes", () => {
    it.each([
      ["float", DataType.INT32, true],
      ["integer", DataType.FLOAT64, true],
      ["unsigned", DataType.INT64, true],
      ["boolean", DataType.UINT8, true],
      ["string", DataType.STRING, true],
      ["string", DataType.FLOAT64, false],
      ["timestamp", DataType.TIMESTAMP, true],
      ["timestamp", DataType.INT64, false],
      ["json", DataType.STRING, false],
    ] as const)("should report whether a %s field takes %s", (kind, dt, expected) => {
      expect(ICD.encodes({ type: kind, scale: 1, offset: 0 }, dt.toString())).toBe(
        expected,
      );
    });

    it.each([
      [DataType.INT64, false],
      [DataType.FLOAT32, true],
    ] as const)(
      "should report whether a scaled integer field takes %s",
      (dt, expected) => {
        expect(ICD.encodes(fieldOf("integer", 0.5), dt.toString())).toBe(expected);
      },
    );
  });

  describe("bindingDataType", () => {
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

    it("should give a binding with an absent data type the default of its field", () => {
      expect(ICD.bindingDataType({}, fieldOf("integer"))).toBe("int64");
    });
  });
});
