// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Only these two code points lower case differently in String.prototype.toLowerCase
// than in the Unicode simple mapping: U+0130 becomes two code points, and U+03A3
// becomes a final sigma at the end of a word.
const SPECIAL_CASING = /[İΣ]/;

const lower = (s: string): string => {
  if (!SPECIAL_CASING.test(s)) return s.toLowerCase();
  let out = "";
  for (const c of s) out += c === "İ" ? "i" : c.toLowerCase();
  return out;
};

// A surrogate encodes a code point above U+FFFF, so it orders after every other code
// unit.
const codePointOrder = (unit: number): number =>
  unit < 0xd800 ? unit : unit >= 0xe000 ? unit - 0x800 : unit + 0x2000;

/**
 * Orders strings by code point, as Go orders strings, rather than by UTF-16 code unit.
 *
 * @returns a negative number if a orders before b, a positive number if after, and 0
 * if they are equal.
 */
export const codePoints = (a: string, b: string): number => {
  const n = Math.min(a.length, b.length);
  for (let i = 0; i < n; i++) {
    const x = a.charCodeAt(i);
    const y = b.charCodeAt(i);
    if (x !== y) return codePointOrder(x) < codePointOrder(y) ? -1 : 1;
  }
  return Math.sign(a.length - b.length);
};

/**
 * Orders text as the Core's sorted indexes do: by the Unicode simple lower case of
 * each string, compared by code point.
 *
 * @returns a negative number if a orders before b, a positive number if after, and 0
 * if they order the same.
 */
export const text = (a: string, b: string): number => codePoints(lower(a), lower(b));
