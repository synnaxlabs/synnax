// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, zod } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Key,
  type Name,
  nameZ,
  type Payload,
  payloadZ,
  type Where,
} from "@/channel/types.gen";
import { NotFoundError } from "@/errors";

/** One or many channels, named by key or by name. */
export type PrimitiveParams = Key | Name | Array<Key | Name>;

/**
 * Zod schema for {@link Params}. A number is a key and a string is a name or a path,
 * digits included. A list may mix them.
 */
export const paramsZ = z.union([
  zod.toArray(z.uint32().or(nameZ)),
  zod.toArray(payloadZ).transform((p) => p.map((c) => c.key)),
]);
/** Anything that names one or many channels. */
export type Params = PrimitiveParams | Payload | Payload[];

const CHAR_REGEX = /[a-zA-Z0-9_]/;

const VALID_NAME_PATTERN = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

/** @returns whether name is a valid channel name. */
export const isValidName = (name: string): boolean => VALID_NAME_PATTERN.test(name);

/**
 * Rewrites a channel name into a valid identifier, replacing every other character with
 * an underscore and prefixing a leading digit.
 */
export const escapeInvalidName = (name: string, changeEmptyToUnderscore = false) => {
  if (name === "") return changeEmptyToUnderscore ? "_" : "";
  if (isValidName(name)) return name;
  // if it doesn't match, convert non-alphanumeric characters to underscores and prepend
  // an underscore if the first character is a digit
  let result = "";
  for (const char of name)
    if (char.match(CHAR_REGEX)) result += char;
    else result += "_";
  if (result[0].match(/^\d/)) result = `_${result}`;
  return result;
};

/** @returns whether a calculation writes the channel. */
export const isCalculated = ({ calculated }: Payload): boolean => calculated;

/** A '.' joins the folders of a channel's path and its name, and no name holds one. */
const PATH_SEPARATOR = ".";

/** A channel named by its key, its name, or its path. */
export type Lookup = Key | Name;

/** @returns each channel of params as a lookup, a payload as its key. */
export const toLookups = (params: Params): Lookup[] =>
  array
    .toArray<Key | Name | Payload>(params)
    .map((p) => (typeof p === "object" ? p.key : p));

/** The lookups of each kind, in the order given. */
export interface SplitLookups {
  keys: Key[];
  names: Name[];
  paths: string[];
}

/**
 * @returns the lookups split by kind. A number is a key, a string that holds a '.' is a
 * path, and any other string is a name, digits included.
 */
export const splitLookups = (lookups: Lookup[]): SplitLookups => {
  const split: SplitLookups = { keys: [], names: [], paths: [] };
  for (const lookup of lookups)
    if (typeof lookup === "number") split.keys.push(lookup);
    else if (lookup.includes(PATH_SEPARATOR)) split.paths.push(lookup);
    else split.names.push(lookup);
  return split;
};

const oneOrIn = <T>(values: T[]): { eq: T } | { in: T[] } =>
  values.length === 1 ? { eq: values[0] } : { in: values };

/**
 * @returns the filter that looks up each channel in lookups, which must not be empty.
 * Lookups of several kinds join in an or. Keys are always a list, which the record
 * store serves.
 */
export const lookupWhere = (lookups: Lookup[]): Where => {
  const { keys, names, paths } = splitLookups(lookups);
  const branches: Where[] = [];
  if (keys.length > 0) branches.push({ key: { in: keys } });
  if (names.length > 0) branches.push({ name: oneOrIn(names) });
  if (paths.length > 0) branches.push({ path: oneOrIn(paths) });
  return branches.length === 1 ? branches[0] : { or: branches };
};

/**
 * Resolves each lookup to its channel through one call of retrieve. A channel matches a
 * lookup by its key, its name in NFC, or its path in NFC.
 * @param retrieve - returns the channels that a list of lookups finds, each with its
 * path when a lookup is a path. It must refuse an ambiguous name.
 * @returns the channel of each lookup, in the order of lookups.
 * @throws {NotFoundError} naming every lookup that finds no channel.
 */
export const resolveLookups = async <C extends { key: Key; name: Name; path?: string }>(
  lookups: Lookup[],
  retrieve: (lookups: Lookup[]) => Promise<C[]>,
): Promise<C[]> => {
  if (lookups.length === 0) return [];
  const named = lookups.some((l) => typeof l === "string");
  const found = new Map<Key | string, C>();
  for (const ch of await retrieve(lookups)) {
    found.set(ch.key, ch);
    if (!named) continue;
    found.set(ch.name, ch);
    if (ch.path != null) found.set(ch.path, ch);
  }
  const missing: Lookup[] = [];
  const resolved: C[] = [];
  for (const lookup of lookups) {
    const ch = found.get(typeof lookup === "number" ? lookup : lookup.normalize("NFC"));
    if (ch == null) missing.push(lookup);
    else resolved.push(ch);
  }
  if (missing.length > 0)
    throw new NotFoundError(`channels not found matching ${JSON.stringify(missing)}`);
  return resolved;
};
