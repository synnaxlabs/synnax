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
  describe("createFields", () => {
    it("should create a timestamp field with the inferred time format", () => {
      const [field] = ICD.createFields([
        { pointer: "/time", kind: "timestamp", timeFormat: "unix_ns" },
      ]);
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
        const [field] = ICD.createFields([{ pointer: "/v", kind }]);
        expect(field).toMatchObject({ name: "v", pointer: "/v", type: kind });
        expect(field).not.toHaveProperty("timeFormat");
      },
    );

    it("should give each field its own key", () => {
      const inferred = { pointer: "/v", kind: "float" } as const;
      const [a, b] = ICD.createFields([inferred, inferred]);
      expect(a.key).not.toEqual(b.key);
    });

    it("should keep the kind of a string field with enum labels", () => {
      const labeled = {
        pointer: "/mode",
        kind: "string" as const,
        enum: ["auto", "manual"],
      };
      const [field] = ICD.createFields([labeled]);
      expect(field).toMatchObject({ pointer: "/mode", type: "string", enumValues: [] });
    });

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

  describe("readKind", () => {
    it.each([
      ["the inferred kind for no labels", [], "string"],
      ["an integer for labels that spell no numbers", ["OFF", "ON"], "integer"],
      ["an integer for labels that spell integers", ["-1", "2", "10"], "integer"],
      ["a float for labels that spell a fraction", ["1", "2.5"], "float"],
      ["an integer for labels that mix numbers and words", ["1", "ON"], "integer"],
    ] as const)("should give %s", (_, labels, kind) => {
      expect(ICD.readKind({ kind: "string", enum: [...labels] })).toEqual(kind);
    });
  });

  describe("createReadField", () => {
    it("should read each enum label as its position in the labels", () => {
      const field = ICD.createReadField(
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

    it.each([
      ["integer", ["1", "2", "10"]],
      ["float", ["1", "2.5"]],
    ] as const)(
      "should read enum labels that spell numbers as a %s field with no enum values",
      (type, labels) => {
        const field = ICD.createReadField(
          { pointer: "/mode", kind: "string", enum: [...labels] },
          "mode",
        );
        expect(field).toMatchObject({
          name: "mode",
          pointer: "/mode",
          type,
          enumValues: [],
        });
      },
    );

    it("should create a field of the inferred kind for no enum labels", () => {
      const field = ICD.createReadField(
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
  });

  describe("createReadFields", () => {
    it("should name each read field after its pointer", () => {
      const fields = ICD.createReadFields([
        { pointer: "/plant/state", kind: "string", enum: ["OFF", "ON"] },
        { pointer: "/speed", kind: "float", enum: [] },
      ]);
      expect(fields).toMatchObject([
        { name: "plant_state", pointer: "/plant/state", type: "integer" },
        { name: "speed", pointer: "/speed", type: "float" },
      ]);
    });
  });

  describe("mergeFields", () => {
    it("should give the field at a pointer the sorted labels of every shape", () => {
      expect(
        ICD.mergeFields([
          { pointer: "/state", kind: "string", enum: ["OFF", "ON"] },
          { pointer: "/speed", kind: "float", enum: [] },
          { pointer: "/state", kind: "string", enum: ["FAULT", "OFF"] },
        ]),
      ).toEqual([
        { pointer: "/state", kind: "string", enum: ["FAULT", "OFF", "ON"] },
        { pointer: "/speed", kind: "float", enum: [] },
      ]);
    });

    it.each([
      ["first", [[], ["OFF", "ON"]]],
      ["second", [["OFF", "ON"], []]],
    ])("should give no labels when the %s shape has none", (_, labels) => {
      const fields = labels.map((l) => ({
        pointer: "/state",
        kind: "string",
        enum: l,
      }));
      expect(ICD.mergeFields(fields)).toEqual([
        { pointer: "/state", kind: "string", enum: [] },
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
    const [field] = ICD.createFields([{ pointer: "/v", kind: "float" }]);

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
