// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { MultiEdit } from "@/platform/multiedit";

describe("MultiEdit", () => {
  describe("fieldsByVariant", () => {
    it("should map each variant to the fields its schema declares", () => {
      const fields = MultiEdit.fieldsByVariant({
        a: z.object({ fillColor: z.string().optional() }),
        b: z.object({ level: z.string() }),
      });
      expect(fields.get("a")).toEqual(new Set(["fillColor"]));
      expect(fields.get("b")).toEqual(new Set(["level"]));
    });

    it("should throw when a schema is not an object", () => {
      expect(() => MultiEdit.fieldsByVariant({ a: z.string() })).toThrow(
        "[multiedit] - schema for a is not an object",
      );
    });
  });

  describe("colorRefs", () => {
    it("should reference each color stored at the fields", () => {
      const fault = uuid.create();
      const refs = MultiEdit.colorRefs(
        "a",
        {
          strokeColor: { type: "named", key: fault },
          fillColor: undefined,
          textColor: { type: "fixed", value: "#00ff00" },
        },
        ["strokeColor", "fillColor", "textColor"],
      );
      expect(refs).toEqual([
        { key: "a", path: "strokeColor", value: { type: "named", key: fault } },
        {
          key: "a",
          path: "textColor",
          value: { type: "fixed", value: color.construct("#00ff00") },
        },
      ]);
    });

    it("should throw when a field holds a plain color", () => {
      expect(() =>
        MultiEdit.colorRefs("a", { fillColor: "#ff0000" }, ["fillColor"]),
      ).toThrow(z.ZodError);
    });
  });

  describe("colorRefs with value fields", () => {
    it("should reference each color value as a value only fixed color", () => {
      const refs = MultiEdit.colorRefs(
        "a",
        { strokeColor: "#ff0000", fillColor: undefined },
        ["strokeColor", "fillColor"],
        { strokeColor: true },
      );
      expect(refs).toEqual([
        {
          key: "a",
          path: "strokeColor",
          value: { type: "fixed", value: color.construct("#ff0000") },
          valueOnly: true,
        },
      ]);
    });

    it("should throw when a value field holds a value that is not a color", () => {
      expect(() =>
        MultiEdit.colorRefs("a", { fillColor: { type: "named" } }, ["fillColor"], {
          fillColor: true,
        }),
      ).toThrow(z.ZodError);
    });
  });
});
