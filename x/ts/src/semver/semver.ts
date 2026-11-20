// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { zod } from "@/zod";

export const versionZ = z
  .string()
  .regex(/^\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$/);

export type Version = z.infer<typeof versionZ>;

export interface CompareOptions {
  /** Whether the comparison skips the major version. */
  majorIgnored?: boolean;
  /** Whether the comparison skips the minor version. */
  minorIgnored?: boolean;
  /** Whether the comparison skips the patch version and the pre-release. */
  patchIgnored?: boolean;
}

type Order = -1 | 0 | 1;

const order = (a: number | string, b: number | string): Order =>
  a < b ? -1 : a > b ? 1 : 0;

const NUMERIC = /^\d+$/;

// A numeric identifier sorts before an alphanumeric one, and a shorter list of
// identifiers sorts before a longer one that it prefixes.
const comparePreRelease = (a: string, b: string): Order => {
  const aParts = a.split(".");
  const bParts = b.split(".");
  for (let i = 0; i < Math.max(aParts.length, bParts.length); i++) {
    const aPart = aParts[i];
    const bPart = bParts[i];
    if (aPart === undefined) return -1;
    if (bPart === undefined) return 1;
    const aNumeric = NUMERIC.test(aPart);
    const bNumeric = NUMERIC.test(bPart);
    if (aNumeric !== bNumeric) return aNumeric ? -1 : 1;
    const result = aNumeric
      ? order(parseInt(aPart, 10), parseInt(bPart, 10))
      : order(aPart, bPart);
    if (result !== 0) return result;
  }
  return 0;
};

interface Parsed {
  parts: [number, number, number];
  preRelease?: string;
}

const parse = (version: Version): Parsed => {
  const valid = zod.parse(versionZ, version, { label: "version" });
  const separator = valid.indexOf("-");
  const main = separator === -1 ? valid : valid.slice(0, separator);
  const [major, minor, patch] = main.split(".").map(Number);
  return {
    // versionZ guarantees three numeric parts.
    parts: [major!, minor!, patch!],
    preRelease: separator === -1 ? undefined : valid.slice(separator + 1),
  };
};

/**
 * Compares two semantic versions by semver precedence.
 *
 * @returns -1 if a is older than b, 0 if they are equal, and 1 if a is newer than b.
 * @throws {zod.ParseError} if either value is not a semantic version.
 */
export const compare = (
  a: Version,
  b: Version,
  options: CompareOptions = {},
): Order => {
  const { majorIgnored = false, minorIgnored = false, patchIgnored = false } = options;
  const aParsed = parse(a);
  const bParsed = parse(b);
  const [aMajor, aMinor, aPatch] = aParsed.parts;
  const [bMajor, bMinor, bPatch] = bParsed.parts;
  const parts: Array<[number, number, boolean]> = [
    [aMajor, bMajor, majorIgnored],
    [aMinor, bMinor, minorIgnored],
    [aPatch, bPatch, patchIgnored],
  ];
  for (const [aPart, bPart, ignored] of parts) {
    if (ignored) continue;
    const result = order(aPart, bPart);
    if (result !== 0) return result;
  }
  if (patchIgnored) return 0;
  const { preRelease: aPreRelease } = aParsed;
  const { preRelease: bPreRelease } = bParsed;
  if (aPreRelease === undefined) return bPreRelease === undefined ? 0 : 1;
  if (bPreRelease === undefined) return -1;
  return comparePreRelease(aPreRelease, bPreRelease);
};

/**
 * @returns true if the two semantic versions are equal.
 * @throws {zod.ParseError} if either value is not a semantic version.
 */
export const equal = (a: Version, b: Version, options?: CompareOptions): boolean =>
  compare(a, b, options) === 0;

/**
 * @returns true if a is newer than b.
 * @throws {zod.ParseError} if either value is not a semantic version.
 */
export const newer = (a: Version, b: Version, options?: CompareOptions): boolean =>
  compare(a, b, options) === 1;

/**
 * @returns true if a is older than b.
 * @throws {zod.ParseError} if either value is not a semantic version.
 */
export const older = (a: Version, b: Version, options?: CompareOptions): boolean =>
  compare(a, b, options) === -1;
