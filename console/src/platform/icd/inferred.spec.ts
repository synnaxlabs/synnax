// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

describe("ICD inferred formats", () => {
  describe("createField", () => {
    it("should create a timestamp field with the inferred time format", () => {
      const field = ICD.createField(
        { pointer: "/time", kind: "timestamp", timeFormat: "unix_ns", enum: [] },
        "time",
      );
      expect(field).toMatchObject({
        name: "time",
        pointer: "/time",
        type: "timestamp",
        timeFormat: "unix_ns",
      });
    });

    it.each(["integer", "unsigned", "float", "boolean", "string"] as const)(
      "should create a %s field with no time format",
      (kind) => {
        const field = ICD.createField({ pointer: "/v", kind, enum: [] }, "v");
        expect(field).toMatchObject({ name: "v", pointer: "/v", type: kind });
        expect(field).not.toHaveProperty("timeFormat");
      },
    );

    it("should create an integer field with a value for each enum label", () => {
      const field = ICD.createField(
        { pointer: "/state", kind: "string", enum: ["OFF", "ON"] },
        "state",
      );
      expect(field).toMatchObject({
        name: "state",
        pointer: "/state",
        type: "integer",
        enumValues: [
          { label: "OFF", value: "0" },
          { label: "ON", value: "1" },
        ],
      });
    });

    it("should give each field its own key", () => {
      const inferred = { pointer: "/v", kind: "float" as const, enum: [] };
      expect(ICD.createField(inferred, "v").key).not.toEqual(
        ICD.createField(inferred, "v").key,
      );
    });
  });

  describe("fieldKind", () => {
    it("should give an integer for a field with enum labels", () => {
      expect(ICD.fieldKind({ kind: "string", enum: ["OFF", "ON"] })).toEqual("integer");
    });

    it("should give the inferred kind for a field with no enum labels", () => {
      expect(ICD.fieldKind({ kind: "string", enum: [] })).toEqual("string");
    });
  });

  describe("createFields", () => {
    it("should name each field after its pointer", () => {
      const fields = ICD.createFields([
        { pointer: "/motor/speed", kind: "float", enum: [] },
        { pointer: "/time", kind: "timestamp", timeFormat: "unix_ns", enum: [] },
      ]);
      expect(fields).toMatchObject([
        { name: "motor_speed", pointer: "/motor/speed", type: "float" },
        { name: "time", pointer: "/time", type: "timestamp", timeFormat: "unix_ns" },
      ]);
    });
  });

  describe("kindName", () => {
    it("should name the message of a kind after its present field", () => {
      expect(ICD.kindName("plant/pump", "/status/rpm")).toEqual(
        "plant/pump status_rpm",
      );
    });

    it("should keep the name of a source with one kind", () => {
      expect(ICD.kindName("plant/pump")).toEqual("plant/pump");
    });
  });

  describe("createMessage", () => {
    const field = ICD.createField({ pointer: "/v", kind: "float", enum: [] }, "v");

    it("should create a JSON message of the fields with the name", () => {
      expect(ICD.createMessage("plant/oven", [field], [])).toMatchObject({
        name: "plant/oven",
        payload: { format: "json", fields: [field] },
      });
    });

    it("should number a name that a taken message holds", () => {
      const first = ICD.createMessage("plant/oven", [], []);
      const second = ICD.createMessage("plant/oven", [], [first]);
      expect(second.name).toEqual("plant/oven 2");
      expect(ICD.createMessage("plant/oven", [], [first, second]).name).toEqual(
        "plant/oven 3",
      );
    });

    it("should name a message with an empty name Untitled", () => {
      expect(ICD.createMessage("", [], []).name).toEqual("Untitled");
    });
  });

  describe("namedBy", () => {
    it.each([
      ["plant/oven", "plant/oven"],
      ["plant/oven 2", "plant/oven"],
      ["Untitled", ""],
      ["Untitled 3", ""],
    ])("should accept %j as a name for %j", (name, base) => {
      expect(ICD.namedBy(name, base)).toBe(true);
    });

    it.each([
      ["Oven", "plant/oven"],
      ["plant/oven 2a", "plant/oven"],
      ["plant/oven2", "plant/oven"],
      ["plant/oven ", "plant/oven"],
      ["Untitled", "plant/oven"],
    ])("should reject %j as a name for %j", (name, base) => {
      expect(ICD.namedBy(name, base)).toBe(false);
    });
  });
});
