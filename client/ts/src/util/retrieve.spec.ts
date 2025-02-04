// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect,it } from "vitest";
import { type Primitive } from "zod";

import {
  analyzeParams,
  type ParamAnalysisResult,
  type PartialTypeNameRecord,
} from "@/util/retrieve";

describe("retrieve utils", () => {
  describe("analyze params", () => {
    interface Spec {
      args: Primitive extends any ? any : never;
      variantMap: PartialTypeNameRecord<Primitive>;
      expected: ParamAnalysisResult<Primitive, PartialTypeNameRecord<Primitive>>;
    }

    const SPECS: Spec[] = [
      {
        args: "abc",
        variantMap: { string: "name" },
        expected: {
          single: true,
          variant: "name",
          normalized: ["abc"],
          actual: "abc",
        },
      },
      {
        args: 123,
        variantMap: { number: "id" },
        expected: {
          single: true,
          variant: "id",
          normalized: [123],
          actual: 123,
        },
      },
    ];

    SPECS.forEach(({ args, variantMap, expected }) => {
      it(`should analyze ${JSON.stringify(args)} with ${JSON.stringify(variantMap)}`, () => {
        expect(analyzeParams(args, variantMap)).toEqual(expected);
      });
    });
  });
});
