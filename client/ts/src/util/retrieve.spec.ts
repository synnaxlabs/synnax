// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type primitive } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { MultipleFoundError, NotFoundError } from "@/errors";
import {
  analyzeParams,
  checkForMultipleOrNoResults,
  type ParamAnalysisResult,
  type PartialTypeNameRecord,
} from "@/util/retrieve";

describe("retrieve utils", () => {
  describe("analyze params", () => {
    interface Spec {
      args: primitive.Value;
      variantMap: PartialTypeNameRecord<primitive.Value>;
      expected: ParamAnalysisResult<
        primitive.Value,
        PartialTypeNameRecord<primitive.Value>
      >;
    }

    const SPECS: Spec[] = [
      {
        args: "abc",
        variantMap: { string: "name" },
        expected: { single: true, variant: "name", normalized: ["abc"], actual: "abc" },
      },
      {
        args: 123,
        variantMap: { number: "id" },
        expected: { single: true, variant: "id", normalized: [123], actual: 123 },
      },
    ];

    SPECS.forEach(({ args, variantMap, expected }) => {
      it(`should analyze ${JSON.stringify(args)} with ${JSON.stringify(variantMap)}`, () => {
        expect(analyzeParams(args, variantMap)).toEqual(expected);
      });
    });
  });

  describe("check for multiple or no results", () => {
    const request = { where: { name: { startsWith: "press" } } };

    it("should accept exactly one result", () => {
      expect(() =>
        checkForMultipleOrNoResults("range", request, [1], true),
      ).not.toThrow();
    });

    it("should check nothing for a retrieve of many", () => {
      expect(() =>
        checkForMultipleOrNoResults("range", request, [], false),
      ).not.toThrow();
    });

    it("should describe the filter when nothing matches", () => {
      expect(() => checkForMultipleOrNoResults("range", request, [], true)).toThrow(
        new NotFoundError('Found no range where name starts with "press"'),
      );
    });

    it("should describe the filter when more than one record matches", () => {
      expect(() => checkForMultipleOrNoResults("range", request, [1, 2], true)).toThrow(
        new MultipleFoundError(
          'Expected one range where name starts with "press", but found 2',
        ),
      );
    });

    it("should name the search term before the filter", () => {
      const searched = { ...request, searchTerm: "tank" };
      expect(() => checkForMultipleOrNoResults("range", searched, [], true)).toThrow(
        new NotFoundError(
          'Found no range matching "tank" where name starts with "press"',
        ),
      );
    });

    it("should name the search term when there is no filter", () => {
      expect(() =>
        checkForMultipleOrNoResults("range", { searchTerm: "tank" }, [1, 2], true),
      ).toThrow(
        new MultipleFoundError('Expected one range matching "tank", but found 2'),
      );
    });

    it("should leave out the filter and search term when both are empty", () => {
      expect(() => checkForMultipleOrNoResults("range", {}, [], true)).toThrow(
        new NotFoundError("Found no range"),
      );
      expect(() =>
        checkForMultipleOrNoResults(
          "range",
          { where: {}, searchTerm: "" },
          [1, 2],
          true,
        ),
      ).toThrow(new MultipleFoundError("Expected one range, but found 2"));
    });
  });
});
