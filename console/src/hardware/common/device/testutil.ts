// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { type z } from "zod";

/**
 * Shared test suite that verifies a device propertiesZ schema is tolerant of
 * missing / incomplete data — the core requirement for backward-compatible
 * device retrieval.
 *
 * @param name - human-readable integration name shown in test output.
 * @param schema - the Zod schema under test (e.g. `propertiesZ`).
 * @param zeroProperties - the ZERO_PROPERTIES constant for the integration.
 * @param partialCases - optional extra cases: each entry is `[label, input]`
 *   where `input` is a partial properties object that the schema must accept.
 */
interface TestPropertiesSchemaOptions {
  /** Set to false to skip the empty `{}` parse test (e.g., for versioned schemas). */
  testEmpty?: boolean;
}

export const testPropertiesSchema = (
  name: string,
  schema: z.ZodType,
  zeroProperties: unknown,
  partialCases: Array<[string, unknown]> = [],
  options: TestPropertiesSchemaOptions = {},
): void => {
  const { testEmpty = true } = options;
  describe(`${name} propertiesZ`, () => {
    it("should parse ZERO_PROPERTIES", () => {
      expect(schema.safeParse(zeroProperties).success).toBe(true);
    });

    if (testEmpty)
      it("should parse completely empty properties", () => {
        expect(schema.safeParse({}).success).toBe(true);
      });

    for (const [label, input] of partialCases)
      it(`should parse ${label}`, () => {
        expect(schema.safeParse(input).success).toBe(true);
      });
  });
};
