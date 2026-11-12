// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const modeZ = z.enum(["text", "regex"]);
export type Mode = z.infer<typeof modeZ>;

export const searchZ = z.object({
  text: z.string().default(""),
  mode: modeZ.default("text"),
});
export interface Search extends z.infer<typeof searchZ> {}

const SPECIAL = /[.*+?^${}()|[\]\\]/g;

/**
 * Returns the pattern of the search, or null when the text is empty or an invalid
 * regex. Plain text ignores case.
 */
export const compile = ({ text, mode }: Search, flags: string = ""): RegExp | null => {
  if (text === "") return null;
  if (mode === "text") return new RegExp(text.replace(SPECIAL, "\\$&"), `i${flags}`);
  try {
    return new RegExp(text, flags);
  } catch {
    return null;
  }
};

/** Returns whether the search is a regex that does not compile. */
export const invalid = (search: Search): boolean =>
  search.text !== "" && compile(search) == null;

/**
 * Returns the start and end of each match of the pattern in the text a log draws for a
 * line. A line matches when the result is not empty. An empty match, such as one of
 * `^` or a lookahead, counts and has the same start and end. The pattern must have the
 * global flag.
 */
export const find = (pattern: RegExp, text: string): Array<[number, number]> =>
  [...text.matchAll(pattern)].map((m) => [m.index, m.index + m[0].length]);
