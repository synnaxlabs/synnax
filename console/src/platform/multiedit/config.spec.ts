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
});
