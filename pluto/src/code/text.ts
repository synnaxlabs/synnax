// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const codePoints = (s: string): string[] => Array.from(s);

/** Diff is a single contiguous text change measured in code points: delete deleteCount
 * code points starting at index, then insert insert in their place. */
export interface Diff {
  index: number;
  deleteCount: number;
  insert: string;
}

/** utf16Offset converts a code-point index into the UTF-16 offset a Monaco model
 * addresses with. */
export const utf16Offset = (s: string, codePointIndex: number): number =>
  codePoints(s).slice(0, codePointIndex).join("").length;

/** diff computes the single contiguous change that turns oldStr into newStr, measured in
 * code points: the longest common prefix and suffix are stripped and what remains in the
 * middle is the deletion (from oldStr) and insertion (from newStr). */
export const diff = (oldStr: string, newStr: string): Diff => {
  const o = codePoints(oldStr);
  const n = codePoints(newStr);
  const max = Math.min(o.length, n.length);
  let prefix = 0;
  while (prefix < max && o[prefix] === n[prefix]) prefix++;
  let suffix = 0;
  while (
    suffix < o.length - prefix &&
    suffix < n.length - prefix &&
    o[o.length - 1 - suffix] === n[n.length - 1 - suffix]
  )
    suffix++;
  return {
    index: prefix,
    deleteCount: o.length - prefix - suffix,
    insert: n.slice(prefix, n.length - suffix).join(""),
  };
};
