// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { zod } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Key,
  keyZ,
  type Name,
  nameZ,
  ontologyID,
  type Payload,
  payloadZ,
} from "@/channel/types.gen";
import { idToString } from "@/ontology/payload";
import {
  analyzeParams as baseAnalyzeParams,
  type ParamAnalysisResult,
} from "@/util/retrieve";

/** One or many channels, named by key or by name. */
export type PrimitiveParams = Key | Name | Key[] | Name[];

/** Zod schema for {@link Params}, resolving every form to a key array. */
export const paramsZ = z.union([
  zod.toArray(keyZ),
  zod.toArray(nameZ),
  zod.toArray(payloadZ).transform((p) => p.map((c) => c.key)),
]);
/** Anything that names one or many channels. */
export type Params = PrimitiveParams | Payload | Payload[];

const CHAR_REGEX = /[a-zA-Z0-9_]/;

const VALID_NAME_PATTERN = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

/**
 * Rewrites a channel name into a valid identifier, replacing every other character with
 * an underscore and prefixing a leading digit.
 */
export const escapeInvalidName = (name: string, changeEmptyToUnderscore = false) => {
  if (name === "") return changeEmptyToUnderscore ? "_" : "";
  if (name.match(VALID_NAME_PATTERN)) return name;
  // if it doesn't match, convert non-alphanumeric characters to underscores and prepend
  // an underscore if the first character is a digit
  let result = "";
  for (const char of name)
    if (char.match(CHAR_REGEX)) result += char;
    else result += "_";
  if (result[0].match(/^\d/)) result = `_${result}`;
  return result;
};

/** @returns the key of the status that reports on the given channel. */
export const statusKey = (channel: Key): string => idToString(ontologyID(channel));

/**
 * Resolves channel params into a normalized array, reporting whether the caller named
 * one channel or many, and whether by key or by name.
 */
export const analyzeParams = (
  channels: Params,
): ParamAnalysisResult<Key | Name, { number: "keys"; string: "names" }> => {
  if (Array.isArray(channels) && channels.length > 0 && typeof channels[0] === "object")
    channels = (channels as Payload[]).map((c) => c.key);
  else if (typeof channels === "object" && "key" in channels) channels = [channels.key];
  return baseAnalyzeParams(channels as PrimitiveParams, {
    number: "keys",
    string: "names",
  });
};
