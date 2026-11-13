// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { http } from "@synnaxlabs/client";
import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  dataTypeOf,
  type FillField,
  formatFill,
  hasFill,
  matchKind,
  parseFill,
  retype,
} from "@/feature/http/task/adapter";
import { type ReadField } from "@/feature/http/task/types";

const BASE = { key: "f1", name: "temp", disabled: true, channel: 0, pointer: "/t" };

const createField = (fields: Record<string, unknown>) =>
  http.readFieldZ.parse({ ...BASE, ...fields });

const createFillField = (fields: Record<string, unknown>): FillField => {
  const field = createField(fields);
  if (!hasFill(field)) throw new Error(`a ${field.type} field takes no fill`);
  return field;
};

describe("adapter", () => {
  describe("dataTypeOf", () => {
    it("should return the data type of a numeric field", () => {
      const field = createField({ type: "integer", dataType: "int16" });
      expect(dataTypeOf(field).toString()).toBe("int16");
    });

    it("should return the data type of the kind with no data type", () => {
      expect(dataTypeOf(createField({ type: "unsigned" })).toString()).toBe("uint64");
      expect(dataTypeOf(createField({ type: "boolean" })).toString()).toBe("boolean");
    });
  });

  describe("retype", () => {
    it("should keep the options of a field of the same kind", () => {
      const field = createField({
        type: "float",
        dataType: "float64",
        fill: 2,
        enumValues: [{ label: "ON", value: 1 }],
      });
      expect(retype(field, DataType.FLOAT32)).toMatchObject({
        type: "float",
        dataType: DataType.FLOAT32,
        fill: 2,
        enumValues: [{ label: "ON", value: 1 }],
      });
    });

    it("should convert the fill and entries between numeric kinds", () => {
      const field = createField({
        type: "float",
        fill: 2,
        enumValues: [
          { label: "ON", value: 1 },
          { label: "HALF", value: 0.5 },
          { label: "BIG", value: 300 },
        ],
      });
      expect(retype(field, DataType.UINT8)).toMatchObject({
        type: "unsigned",
        fill: "2",
        enumValues: [{ label: "ON", value: "1" }],
      });
    });

    it("should drop an integer fill that a float has no equal for", () => {
      const field = createField({ type: "integer", fill: "9007199254740993" });
      const float = retype(field, DataType.FLOAT64);
      expect(float.type).toBe("float");
      expect("fill" in float ? float.fill : undefined).toBeUndefined();
    });

    it("should set the data type of a numeric field that has none", () => {
      const field = createField({ type: "float" });
      expect(retype(field, DataType.FLOAT32)).toMatchObject({
        type: "float",
        dataType: DataType.FLOAT32,
      });
    });

    it("should drop a non-finite float fill held as text on an integer kind", () => {
      // The cache holds the Core's text for a non-finite float.
      const field = {
        ...createField({ type: "float" }),
        fill: "Infinity",
      } as unknown as ReadField;
      const integer = retype(field, DataType.INT32);
      expect(integer.type).toBe("integer");
      expect("fill" in integer ? integer.fill : undefined).toBeUndefined();
    });

    it("should make numeric entries and a fill booleans on a boolean kind", () => {
      const field = createField({
        type: "float",
        fill: 1,
        enumValues: [
          { label: "ON", value: 1 },
          { label: "OFF", value: 0 },
          { label: "UNKNOWN", value: NaN },
        ],
      });
      expect(retype(field, DataType.BOOLEAN)).toMatchObject({
        type: "boolean",
        fill: true,
        enumValues: [
          { label: "ON", value: true },
          { label: "OFF", value: false },
        ],
      });
    });

    it("should drop a fill and entries that a narrower data type cannot hold", () => {
      const field = createField({
        type: "integer",
        dataType: "int64",
        fill: "1000",
        enumValues: [
          { label: "LOW", value: "-5" },
          { label: "HIGH", value: "300" },
        ],
      });
      const integer = retype(field, DataType.INT8);
      expect(integer).toMatchObject({
        type: "integer",
        dataType: DataType.INT8,
        enumValues: [{ label: "LOW", value: "-5" }],
      });
      expect("fill" in integer ? integer.fill : undefined).toBeUndefined();
    });

    it("should keep only the base fields of a field of another kind", () => {
      const field = createField({ type: "float", fill: 2 });
      expect(retype(field, DataType.STRING)).toEqual({
        ...BASE,
        type: "string",
        enumValues: [],
      });
    });

    it("should give a new timestamp field a time format", () => {
      const field = createField({ type: "float" });
      expect(retype(field, DataType.TIMESTAMP)).toEqual({
        ...BASE,
        type: "timestamp",
        timeFormat: "iso8601",
      });
    });
  });

  describe("matchKind", () => {
    it.each([
      [DataType.FLOAT32, "float"],
      [DataType.INT16, "integer"],
      [DataType.UINT8, "unsigned"],
      [DataType.BOOLEAN, "boolean"],
      [DataType.STRING, "string"],
      [DataType.TIMESTAMP, "timestamp"],
    ])("should give a field on a %s channel the %s kind", (dataType, kind) => {
      expect(matchKind(createField({ type: "json" }), dataType).type).toBe(kind);
    });

    it("should keep a field with no channel data type", () => {
      const field = createField({ type: "float", fill: 2 });
      expect(matchKind(field)).toBe(field);
    });

    it("should keep a field of the channel's kind", () => {
      const field = createField({ type: "float", fill: 2 });
      expect(matchKind(field, DataType.FLOAT32)).toBe(field);
    });

    it("should retype a field of another kind", () => {
      const field = createField({ type: "float", fill: 2 });
      expect(matchKind(field, DataType.STRING)).toEqual({
        ...BASE,
        type: "string",
        enumValues: [],
      });
    });
  });

  describe("formatFill", () => {
    it("should show a NaN float fill as empty", () => {
      expect(formatFill(createFillField({ type: "float", fill: "NaN" }))).toBe("");
    });

    it("should show an infinite float fill as its text", () => {
      const field = createFillField({ type: "float", fill: "Infinity" });
      expect(formatFill(field)).toBe("Infinity");
    });

    it("should show a boolean fill as its text", () => {
      expect(formatFill(createFillField({ type: "boolean", fill: false }))).toBe(
        "false",
      );
    });
  });

  describe("parseFill", () => {
    it.each([
      ["float", "", NaN],
      ["float", "-Infinity", -Infinity],
      ["float", "1.5", 1.5],
      ["integer", "-9223372036854775808", "-9223372036854775808"],
      ["integer", "", undefined],
      ["unsigned", "18446744073709551615", "18446744073709551615"],
      ["boolean", "true", true],
      ["string", " on ", " on "],
      ["string", "", undefined],
      ["string", " ", " "],
    ])("should parse a %s fill of %j", (type, text, fill) => {
      expect(parseFill(createFillField({ type }), text)?.fill).toEqual(fill);
    });

    it.each([
      ["float", "nan"],
      ["float", "off"],
      ["integer", "1.5"],
      ["integer", "9223372036854775808"],
      ["unsigned", "-1"],
      ["unsigned", "18446744073709551616"],
      ["boolean", "1"],
    ])("should reject a %s fill of %j", (type, text) => {
      expect(parseFill(createFillField({ type }), text)).toBeNull();
    });
  });
});
