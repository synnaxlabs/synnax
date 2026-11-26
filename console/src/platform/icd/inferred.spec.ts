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
        { pointer: "/time", kind: "timestamp", timeFormat: "unix_ns" },
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
        const field = ICD.createField({ pointer: "/v", kind }, "v");
        expect(field).toMatchObject({ name: "v", pointer: "/v", type: kind });
        expect(field).not.toHaveProperty("timeFormat");
      },
    );

    it("should give each field its own key", () => {
      const inferred = { pointer: "/v", kind: "float" } as const;
      expect(ICD.createField(inferred, "v").key).not.toEqual(
        ICD.createField(inferred, "v").key,
      );
    });
  });

  describe("createFields", () => {
    it("should name each field after its pointer", () => {
      const fields = ICD.createFields([
        { pointer: "/motor/speed", kind: "float" },
        { pointer: "/time", kind: "timestamp", timeFormat: "unix_ns" },
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
    const field = ICD.createField({ pointer: "/v", kind: "float" }, "v");

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
