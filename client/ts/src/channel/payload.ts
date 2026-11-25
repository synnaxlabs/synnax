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
  type Name,
  nameZ,
  type Payload,
  payloadZ,
  type Where,
} from "@/channel/types.gen";
import { util } from "@/util";

/** One or many channels, named by key or by name. */
export type PrimitiveParams = Key | Name | Key[] | Name[];

/**
 * Zod schema for {@link Params}. A number is a key and a string is a name, digits
 * included.
 */
export const paramsZ = z.union([
  zod.toArray(z.uint32()),
  zod.toArray(nameZ),
  zod.toArray(payloadZ).transform((p) => p.map((c) => c.key)),
]);
/** Anything that names one or many channels. */
export type Params = PrimitiveParams | Payload | Payload[];

// REFUSED matches a character that a name cannot hold: the path separator, and a
// character that shows as nothing or as a space other than U+0020.
const REFUSED =
  /[.\p{Cc}\p{Cf}\p{Zl}\p{Zp}\p{Default_Ignorable_Code_Point}\u2800]|(?! )\p{Zs}/u;
const REFUSED_ALL = new RegExp(REFUSED.source, "gu");
const VISIBLE = /[^ ]/;

/**
 * @returns whether name is a valid channel name: in NFC, with a character other than a
 * space, and with no '.' and no character that shows as nothing or as a space other
 * than U+0020.
 */
export const isValidName = (name: string): boolean =>
  VISIBLE.test(name) && !REFUSED.test(name) && name.normalize("NFC") === name;

/**
 * Rewrites a channel name into a valid one: in NFC, with an underscore in place of each
 * character that a name cannot hold and of each space of a name of spaces alone.
 */
export const escapeInvalidName = (name: string, changeEmptyToUnderscore = false) => {
  if (name === "") return changeEmptyToUnderscore ? "_" : "";
  const escaped = name.normalize("NFC").replace(REFUSED_ALL, "_");
  return VISIBLE.test(escaped) ? escaped : escaped.replaceAll(" ", "_");
};

/** @returns whether the channel computes its samples from an expression. */
export const isCalculated = ({ virtual, expression }: Payload): boolean =>
  virtual && expression !== "";

/**
 * Resolves channel params into a normalized array, reporting whether the caller named
 * one channel or many, and whether by key or by name.
 */
export const analyzeParams = (
  channels: Params,
): util.ParamAnalysisResult<Key | Name, { number: "keys"; string: "names" }> => {
  if (Array.isArray(channels) && channels.length > 0 && typeof channels[0] === "object")
    channels = (channels as Payload[]).map((c) => c.key);
  else if (typeof channels === "object" && "key" in channels) channels = [channels.key];
  return util.analyzeParams(
    channels as PrimitiveParams,
    { number: "keys", string: "names" },
    { convertNumericStrings: false },
  );
};

/**
 * @returns the filter that a retrieve of one channel sends. A number is a key and a
 * string is a name.
 */
export const singleWhere = (channel: Key | Name): Where => {
  const { variant, normalized } = analyzeParams(channel);
  return variant === "keys"
    ? { key: { eq: normalized[0] } }
    : { name: { eq: normalized[0] } };
};
