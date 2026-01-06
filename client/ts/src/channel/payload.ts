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

import { type Key, keyZ, type Payload, payloadZ } from "@/channel/types.gen";

// const errorMessage = "Channel key must be a valid uint32.";
// export const keyZ = z.uint32().or(
//   z
//     .string()
//     .refine((val) => !isNaN(Number(val)), { message: errorMessage })
//     .transform(Number)
//     .refine((val) => val < math.MAX_UINT32, { message: errorMessage }),
// );
export type PrimitiveParams = Key | Name | Keys | Names;
const VALID_NAME_PATTERN = /^[a-zA-Z_][a-zA-Z0-9_]*$/;
export const nameZ = z
  .string()
  .min(1, "Name must not be empty")
  .regex(
    VALID_NAME_PATTERN,
    "Name can only contain letters, digits, and underscores, and cannot start with a digit",
  );
export type Name = z.infer<typeof nameZ>;

export const paramsZ = z.union([
  zod.toArray(keyZ),
  zod.toArray(nameZ),
  zod.toArray(payloadZ).transform((p) => p.map((c) => c.key)),
]);
export type Keys = Key[];
export type Names = Name[];
export type Payloads = Payload[];
export type KeysOrNames = Key | Name | Keys | Names;
export type Params = Key | Name | Keys | Names | Payload | Payload[];
export type KeyOrName = Key | Name;

const CHAR_REGEX = /[a-zA-Z0-9_]/;

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
