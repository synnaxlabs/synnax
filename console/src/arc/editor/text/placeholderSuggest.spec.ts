// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { shouldTriggerSuggestion } from "@/arc/editor/text/placeholderSuggest";

interface TestCase {
  name: string;
  buffer: string;
  typed: string;
  expected: boolean;
}

const TEST_CASES: TestCase[] = [
  {
    name: "inside open backtick + open brace",
    buffer: "`hello {na",
    typed: "m",
    expected: true,
  },
  {
    name: "after closed brace, no longer in placeholder",
    buffer: "`hello {n}",
    typed: "o",
    expected: false,
  },
  {
    name: "inside backtick but no open brace",
    buffer: "`hello ",
    typed: "o",
    expected: false,
  },
  {
    name: "nested open brace counts as placeholder",
    buffer: "`a + {b + {c",
    typed: "d",
    expected: true,
  },
  {
    name: "typed char is `}` (not a word char)",
    buffer: "`hello {x",
    typed: "}",
    expected: false,
  },
  { name: "digit is a word char", buffer: "`hello {x", typed: "1", expected: true },
  {
    name: "underscore is a word char",
    buffer: "`hello {x",
    typed: "_",
    expected: true,
  },
  {
    name: "double-quoted string is not a backtick raw string",
    buffer: '"{x',
    typed: "y",
    expected: false,
  },
  { name: "empty buffer never matches", buffer: "", typed: "a", expected: false },
  {
    name: "typed empty string never fires",
    buffer: "`hello {x",
    typed: "",
    expected: false,
  },
  {
    name: "multi-char typed input never fires (paste)",
    buffer: "`hello {x",
    typed: "ab",
    expected: false,
  },
  {
    name: "space is not a word char",
    buffer: "`hello {x",
    typed: " ",
    expected: false,
  },
  {
    name: "caret on second line inside placeholder",
    buffer: "x = `hi\n{na",
    typed: "m",
    expected: true,
  },
  {
    name: "closed placeholder followed by new open one",
    buffer: "`{a} {b",
    typed: "c",
    expected: true,
  },
  {
    name: "escaped open brace does not trigger",
    buffer: "`hello \\{va",
    typed: "l",
    expected: false,
  },
  {
    name: "real placeholder after escaped brace still triggers",
    buffer: "`\\{ {va",
    typed: "l",
    expected: true,
  },
];

describe("shouldTriggerSuggestion", () => {
  TEST_CASES.forEach(({ name, buffer, typed, expected }) =>
    it(name, () => {
      expect(shouldTriggerSuggestion(buffer, typed)).toBe(expected);
    }),
  );
});
