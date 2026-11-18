// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { MultiEdit } from "@/platform/multiedit";

describe("MultiEdit", () => {
  describe("fixedColorRefs", () => {
    it("should reference each color stored at the fields as a fixed color", () => {
      const refs = MultiEdit.fixedColorRefs(
        "a",
        { strokeColor: "#ff0000", fillColor: undefined, textColor: "#00ff00" },
        ["strokeColor", "fillColor", "textColor"],
      );
      expect(refs).toEqual([
        {
          key: "a",
          path: "strokeColor",
          value: { type: "fixed", value: color.construct("#ff0000") },
        },
        {
          key: "a",
          path: "textColor",
          value: { type: "fixed", value: color.construct("#00ff00") },
        },
      ]);
    });

    it("should throw when a field holds a value that is not a color", () => {
      expect(() =>
        MultiEdit.fixedColorRefs("a", { fillColor: { type: "named" } }, ["fillColor"]),
      ).toThrow(z.ZodError);
    });
  });

  describe("fixedColor", () => {
    it("should return the color of a fixed reference", () => {
      expect(MultiEdit.fixedColor(MultiEdit.fixedRef("#ff0000"))).toEqual(
        color.construct("#ff0000"),
      );
    });

    it("should throw for a named reference", () => {
      expect(() => MultiEdit.fixedColor({ type: "named", key: "nominal" })).toThrow(
        "[multiedit] - named color nominal in a fixed color field",
      );
    });
  });
});
