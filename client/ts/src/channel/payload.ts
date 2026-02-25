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
  type Payload,
  payloadZ,
} from "@/channel/types.gen";

export type Keys = Key[];
export type KeyOrName = Key | Name;
export type Names = Name[];
export type Payloads = Payload[];
export type PrimitiveParams = Key | Keys | Names | Name;
export type KeysOrNames = Keys | Names;

export const paramsZ = z.union([
  zod.toArray(keyZ),
  zod.toArray(nameZ),
  zod.toArray(payloadZ).transform((p) => p.map((c) => c.key)),
]);
export type Params = PrimitiveParams | Payload | Payload[];

const CHAR_REGEX = /[a-zA-Z0-9_]/;

const VALID_NAME_PATTERN = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

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
