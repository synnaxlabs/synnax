// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { text } from "@/text/aether";

// Every code point is one unit wide, so a grapheme made of several code points is wider
// than one unit and a cut through it would show.
const measure = (label: string): number => Array.from(label).length;

describe("truncate", () => {
  it("should return text that fits unchanged", () => {
    expect(text.truncate("hello", 5, measure)).toBe("hello");
  });

  it("should cut text that overflows and end it with an ellipsis", () => {
    expect(text.truncate("hello world", 6, measure)).toBe("hello…");
  });

  it("should keep the longest start of the text that fits", () => {
    expect(text.truncate("abcdefghij", 4, measure)).toBe("abc…");
  });

  it("should return only the ellipsis when no character fits beside it", () => {
    expect(text.truncate("hello", 1, measure)).toBe("…");
  });

  it("should return an empty string when the ellipsis does not fit", () => {
    expect(text.truncate("hello", 0.5, measure)).toBe("");
  });

  it("should return an empty string for empty text", () => {
    expect(text.truncate("", 0, measure)).toBe("");
  });

  it("should never split an emoji made of several code points", () => {
    const family = "👨‍👩‍👧";
    expect(text.truncate(`ab${family}cd`, 7, measure)).toBe("ab…");
  });

  it("should never split a letter from its combining accent", () => {
    expect(text.truncate("café au lait", 5, measure)).toBe("caf…");
  });

  it("should measure a logarithmic number of candidates", () => {
    const spy = vi.fn(measure);
    text.truncate("x".repeat(1024), 100, spy);
    expect(spy.mock.calls.length).toBeLessThanOrEqual(14);
  });
});
