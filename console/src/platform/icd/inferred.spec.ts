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

describe("ICD inferred fields", () => {
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
});
