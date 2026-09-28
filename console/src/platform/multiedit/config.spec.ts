// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

  describe("patch", () => {
    it("should set nested paths on a copy", () => {
      const config = { label: { level: "p" }, scale: 1 };
      const next = MultiEdit.patch(config, [
        ["label.level", "h5"],
        ["scale", 2],
      ]);
      expect(next).toEqual({ label: { level: "h5" }, scale: 2 });
      expect(config).toEqual({ label: { level: "p" }, scale: 1 });
    });

    it("should remove a path set to undefined", () => {
      const next = MultiEdit.patch({ fillColor: "#ff0000", scale: 1 }, [
        ["fillColor", undefined],
      ]);
      expect(next).toEqual({ scale: 1 });
    });
  });

  describe("groupByColor", () => {
    it("should group references by color in first-seen order", () => {
      const refs = [
        ...MultiEdit.colorRefs("a", { strokeColor: "#ff0000" }, ["strokeColor"]),
        ...MultiEdit.colorRefs("b", { fillColor: "#00ff00" }, [
          "strokeColor",
          "fillColor",
        ]),
        ...MultiEdit.colorRefs("c", { textColor: "#ff0000" }, ["textColor"]),
      ];
      const groups = MultiEdit.groupByColor(refs);
      expect(Array.from(groups.keys())).toEqual(["#ff0000", "#00ff00"]);
      expect(groups.get("#ff0000")?.map((r) => `${r.key}.${r.path}`)).toEqual([
        "a.strokeColor",
        "c.textColor",
      ]);
    });
  });
});
