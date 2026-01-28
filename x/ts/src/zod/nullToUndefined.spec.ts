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

import { zod } from "@/zod";

describe("zod", () => {
  describe("nullToUndefined", () => {
    const schema = zod.nullToUndefined(z.string());
    it("should parse the normal case", () => {
      expect(schema.parse("string")).toBe("string");
    });
    it("should parse null as undefined", () => {
      expect(schema.parse(null)).toBeUndefined();
    });
    it("should parse undefined as undefined", () => {
      expect(schema.parse(undefined)).toBeUndefined();
    });
    it("should throw for other values", () => {
      expect(() => schema.parse(1)).toThrow(z.ZodError);
    });
  });
});
