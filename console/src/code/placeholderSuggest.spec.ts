// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { shouldTriggerSuggestion } from "@/code/placeholderSuggest";

describe("shouldTriggerSuggestion", () => {
  it.each([
    ["inside open backtick + open brace", "`hello {na", "m", true],
    ["after closed brace, no longer in placeholder", "`hello {n}", "o", false],
    ["inside backtick but no open brace", "`hello ", "o", false],
    ["nested open brace counts as placeholder", "`a + {b + {c", "d", true],
    ["typed char is `}` (not a word char)", "`hello {x", "}", false],
    ["digit is a word char", "`hello {x", "1", true],
    ["underscore is a word char", "`hello {x", "_", true],
    ["double-quoted string is not a backtick raw string", '"{x', "y", false],
    ["empty buffer never matches", "", "a", false],
    ["typed empty string never fires", "`hello {x", "", false],
    ["multi-char typed input never fires (paste)", "`hello {x", "ab", false],
    ["space is not a word char", "`hello {x", " ", false],
    ["caret on second line inside placeholder", "x = `hi\n{na", "m", true],
    ["closed placeholder followed by new open one", "`{a} {b", "c", true],
    ["escaped open brace does not trigger", "`hello \\{va", "l", false],
    ["real placeholder after escaped brace still triggers", "`\\{ {va", "l", true],
  ])("%s", (_label, buffer, typed, expected) => {
    expect(shouldTriggerSuggestion(buffer, typed)).toBe(expected);
  });
});
