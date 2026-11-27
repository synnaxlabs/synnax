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

const jsonFieldOf = (fields: Record<string, unknown>) =>
  icd.jsonFieldZ.parse({ key: uuid.create(), name: "j", pointer: "/j", ...fields });

const textField = icd.textFieldZ.parse({
  key: uuid.create(),
  name: "t",
  encoding: "delimited",
  position: 1,
});

const binaryField = icd.binaryFieldZ.parse({
  key: uuid.create(),
  name: "b",
  kind: "unsigned",
  bitLength: 16,
});

const timestamp = jsonFieldOf({ type: "timestamp", timeFormat: "unix_ms" });

const level = icd.jsonFieldZ.parse({
  key: uuid.create(),
  name: "level",
  pointer: "/level",
  type: "float",
  scale: 1,
  offset: 0,
});

describe("ICD", () => {
  describe("reindex", () => {
    it("should bind the new index field to the index channel with no data type", () => {
      const message = uuid.create();
      expect(ICD.reindex({ message, bindings: [] }, level.key, 9)).toEqual({
        message,
        index: level.key,
        bindings: [{ field: level.key, channel: 9 }],
      });
    });
  });

  describe("mostRead", () => {
    it("should return undefined when no source reads an ICD", () => {
      expect(ICD.mostRead([])).toBeUndefined();
    });

    it("should return the ICD that the most sources read", () => {
      expect(ICD.mostRead([{ icd: "a" }, { icd: "b" }, { icd: "b" }])).toBe("b");
    });

    it("should return the first ICD that a source reads when counts tie", () => {
      expect(
        ICD.mostRead([{ icd: "a" }, { icd: "b" }, { icd: "b" }, { icd: "a" }]),
      ).toBe("a");
    });
  });

  describe("isReadable", () => {
    it.each([
      [
        "a JSON bytes field with no pointer",
        jsonFieldOf({ type: "bytes", pointer: "" }),
        false,
      ],
      ["a JSON bytes field with a pointer", jsonFieldOf({ type: "bytes" }), true],
      ["a text field", textField, true],
      ["a binary field", binaryField, true],
    ] as const)("should report whether the Core reads %s", (_, field, expected) => {
      expect(ICD.isReadable(field)).toBe(expected);
      expect(ICD.treeFieldOf(field).readable).toBe(expected);
    });
  });

  describe("isIndexable", () => {
    it("should report that a read cannot index on no field", () => {
      expect(ICD.isIndexable(undefined)).toBe(false);
    });
  });

  describe("suit", () => {
    it.each([
      [
        "keep a data type that the field converts to",
        DataType.FLOAT64.toString(),
        true,
      ],
      [
        "drop a data type that the field cannot convert to",
        DataType.INT64.toString(),
        false,
      ],
    ] as const)("should %s", (_, dataType, kept) => {
      const binding = { field: level.key, channel: 7, dataType };
      const suited = ICD.suit({ message: uuid.create(), bindings: [binding] }, [level]);
      expect(suited.bindings).toEqual([
        kept ? binding : { field: level.key, channel: 7 },
      ]);
    });

    it("should keep a binding with no data type", () => {
      const binding = { field: level.key, channel: 7 };
      const suited = ICD.suit({ message: uuid.create(), bindings: [binding] }, [level]);
      expect(suited.bindings).toEqual([binding]);
    });

    const message = uuid.create();

    it("should drop the binding of a field that the message no longer holds", () => {
      const read = {
        message,
        bindings: [
          { field: textField.key, channel: 1 },
          { field: uuid.create(), channel: 2 },
        ],
      };
      expect(ICD.suit(read, [textField]).bindings).toEqual([read.bindings[0]]);
    });

    it("should clear a data type that the field of a binding cannot read into", () => {
      const read = {
        message,
        bindings: [
          { field: textField.key, channel: 1, dataType: "int64" },
          { field: binaryField.key, channel: 2, dataType: "int16" },
        ],
      };
      expect(ICD.suit(read, [textField, binaryField]).bindings).toEqual([
        { field: textField.key, channel: 1 },
        { field: binaryField.key, channel: 2 },
      ]);
    });

    it("should drop an index on a field that is not a timestamp", () => {
      const read = {
        message,
        index: textField.key,
        bindings: [{ field: textField.key, channel: 1 }],
      };
      expect(ICD.suit(read, [textField])).toEqual({ message, bindings: [] });
    });

    it("should keep an index on a JSON timestamp field", () => {
      const read = {
        message,
        index: timestamp.key,
        bindings: [{ field: timestamp.key, channel: 1 }],
      };
      expect(ICD.suit(read, [timestamp])).toEqual(read);
    });
  });
});
