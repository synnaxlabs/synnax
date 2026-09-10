// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { framer } from "@/framer";

const MAX_DOWNSAMPLE_FACTOR = 2 ** 32 - 1;

describe("framer downsample factor", () => {
  describe("iteratorConfigZ", () => {
    test("defaults to keeping every sample", () => {
      expect(framer.iteratorConfigZ.parse({}).downsampleFactor).toEqual(1);
    });
    test.each([0, 1, 2, 10, MAX_DOWNSAMPLE_FACTOR])("accepts %i", (factor) => {
      expect(framer.iteratorConfigZ.parse({ downsampleFactor: factor })).toHaveProperty(
        "downsampleFactor",
        factor,
      );
    });
    // The wire field is unsigned, so a value the schema lets through would be
    // reinterpreted by the Core rather than rejected.
    test.each([-1, 1.5, MAX_DOWNSAMPLE_FACTOR + 1])("rejects %s", (factor) => {
      expect(() =>
        framer.iteratorConfigZ.parse({ downsampleFactor: factor }),
      ).toThrow();
    });
  });
});
