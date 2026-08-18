// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { shouldTriggerSuggestion } from "@/arc/text/placeholderSuggest";

interface TestCase {
  name: string;
  buffer: string;
  typed: string;
  expected: boolean;
}

const TEST_CASES: TestCase[] = [
  {
    name: "inside open f-string + open brace",
    buffer: 'f"hello {na',
    typed: "m",
    expected: true,
  },
  {
    name: "after closed brace, no longer in placeholder",
    buffer: 'f"hello {n}',
    typed: "o",
    expected: false,
  },
  {
    name: "inside f-string but no open brace",
    buffer: 'f"hello ',
    typed: "o",
    expected: false,
  },
  {
    name: "nested open brace counts as placeholder",
    buffer: 'f"a + {b + {c',
    typed: "d",
    expected: true,
  },
  {
    name: "typed char is `}` (not a word char)",
    buffer: 'f"hello {x',
    typed: "}",
    expected: false,
  },
  { name: "digit is a word char", buffer: 'f"hello {x', typed: "1", expected: true },
  {
    name: "underscore is a word char",
    buffer: 'f"hello {x',
    typed: "_",
    expected: true,
  },
  {
    name: "plain double-quoted string is not a format string",
    buffer: '"{x',
    typed: "y",
    expected: false,
  },
  { name: "empty buffer never matches", buffer: "", typed: "a", expected: false },
  {
    name: "typed empty string never fires",
    buffer: 'f"hello {x',
    typed: "",
    expected: false,
  },
  {
    name: "multi-char typed input never fires (paste)",
    buffer: 'f"hello {x',
    typed: "ab",
    expected: false,
  },
  {
    name: "space is not a word char",
    buffer: 'f"hello {x',
    typed: " ",
    expected: false,
  },
  {
    name: "caret on second line inside placeholder",
    buffer: "x = f`hi\n{na",
    typed: "m",
    expected: true,
  },
  {
    name: "closed placeholder followed by new open one",
    buffer: 'f"{a} {b',
    typed: "c",
    expected: true,
  },
  {
    name: "doubled open brace does not trigger",
    buffer: 'f"hello {{va',
    typed: "l",
    expected: false,
  },
  {
    name: "real placeholder after doubled brace still triggers",
    buffer: 'f"{{ {va',
    typed: "l",
    expected: true,
  },
  {
    name: "backslash adjacent to placeholder triggers",
    buffer: 'rf"C:\\path\\{na',
    typed: "m",
    expected: true,
  },
  {
    name: "rf-prefixed string triggers",
    buffer: 'rf"path: {va',
    typed: "l",
    expected: true,
  },
  {
    name: "fr-prefixed string triggers",
    buffer: 'fr"path: {va',
    typed: "l",
    expected: true,
  },
  {
    name: "backtick f-string triggers",
    buffer: "f`report:\n  {va",
    typed: "l",
    expected: true,
  },
  {
    name: "r-prefixed (non-format) string does not trigger",
    buffer: 'r"path {va',
    typed: "l",
    expected: false,
  },
];

describe("shouldTriggerSuggestion", () => {
  TEST_CASES.forEach(({ name, buffer, typed, expected }) =>
    it(name, () => {
      expect(shouldTriggerSuggestion(buffer, typed)).toBe(expected);
    }),
  );
});
