// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** The character joining the parts of a deep path. */
export const SEPARATOR = ".";

/**
 * Rewrites each part of a path. A replacer may return one part, several, or undefined
 * to drop the part.
 */
export const transformPath = (
  path: string,
  replacer: (
    part: string,
    index: number,
    parts: string[],
  ) => string | string[] | undefined,
  separator: string = SEPARATOR,
): string => {
  const parts = path.split(separator);
  const result = parts
    .map((part, index) => {
      const r = replacer(part, index, parts);
      if (r == null) return null;
      if (typeof r === "string") return r;
      return r.join(separator);
    })
    .filter((part) => part != null);
  return result.join(separator);
};

/** @returns one part of a path. A negative index counts back from the end. */
export const element = (path: string, index: number): string => {
  const parts = path.split(SEPARATOR);
  if (index < 0) return parts[parts.length + index];
  return parts[index];
};

/**
 * @returns whether the path starts with the pattern, where `*` matches any one part.
 * An empty pattern matches everything.
 */
export const pathsMatch = (path: string, pattern: string): boolean => {
  if (pattern.length === 0) return true;
  const parts = path.split(SEPARATOR);
  const patterns = pattern.split(SEPARATOR);
  if (patterns.length > parts.length) return false;
  for (let i = 0; i < patterns.length; i++) {
    const part = parts[i];
    const pattern = patterns[i];
    if (pattern === "*") continue;
    if (part !== pattern) return false;
  }
  return true;
};
