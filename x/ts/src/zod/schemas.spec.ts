// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { zod } from "@/zod";

describe("zod", () => {
  describe("uint12", () => {
    describe("success", () => {
      interface Spec {
        input: number;
        expected: number;
      }
      const specs: Spec[] = [
        { input: 0, expected: 0 },
        { input: 1, expected: 1 },
        { input: 2 ** 12 - 1, expected: 2 ** 12 - 1 },
      ];
      specs.forEach(({ input, expected }) => {
        it(`should parse ${input} as ${expected}`, () => {
          const result = zod.uint12Z.parse(input);
          expect(result).toBe(expected);
        });
      });
    });
    describe("failure", () => {
      interface Spec {
        input: unknown;
        expected: string;
      }
      const specs: Spec[] = [
        { input: -1, expected: "Too small: expected number to be >=0" },
        { input: 2 ** 12, expected: "Too big: expected number to be <=4095" },
        { input: "1", expected: "Invalid input: expected number, received string" },
        { input: 1.5, expected: "Invalid input: expected int, received number" },
      ];
      specs.forEach(({ input, expected }) => {
        it(`should throw for ${String(input)}`, () => {
          expect(() => zod.uint12Z.parse(input)).toThrow(expected);
        });
      });
    });
  });
});
