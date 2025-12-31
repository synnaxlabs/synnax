// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  array,
  type CrudeDataType,
  DataType,
  math,
  status,
  TimeSpan,
  zod,
} from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";

const errorMessage = "Channel key must be a valid uint32.";
export const keyZ = z.uint32().or(
  z
    .string()
    .refine((val) => !isNaN(Number(val)), { message: errorMessage })
    .transform(Number)
    .refine((val) => val < math.MAX_UINT32, { message: errorMessage }),
);
export type Key = z.infer<typeof keyZ>;
export type Keys = Key[];
const VALID_NAME_PATTERN = /^[a-zA-Z_][a-zA-Z0-9_]*$/;
export const nameZ = z
  .string()
  .min(1, "Name must not be empty")
  .regex(
    VALID_NAME_PATTERN,
    "Name can only contain letters, digits, and underscores, and cannot start with a digit",
  );
export type Name = z.infer<typeof nameZ>;
export type Names = Name[];
export type KeyOrName = Key | Name;
export type KeysOrNames = Keys | Names;
export type PrimitiveParams = Key | Name | Keys | Names;

export const OPERATION_TYPES = ["min", "max", "avg", "none"] as const;
export const operationType = z.enum(OPERATION_TYPES);
export type OperationType = z.infer<typeof operationType>;

export const operationZ = z.object({
  type: operationType,
  resetChannel: keyZ.optional(),
  duration: TimeSpan.z.optional(),
});

export type Operation = z.infer<typeof operationZ>;

export const statusZ = status.statusZ();
export type Status = z.infer<typeof statusZ>;
export const payloadZ = z.object({
  name: z.string(),
  key: keyZ,
  dataType: DataType.z,
  leaseholder: zod.uint12,
  index: keyZ,
  isIndex: z.boolean(),
  internal: z.boolean(),
  virtual: z.boolean(),
  alias: z.string().optional(),
  expression: z.string().default(""),
  status: statusZ.optional(),
  operations: array.nullishToEmpty(operationZ),
});
export interface Payload extends z.infer<typeof payloadZ> {}

export const newZ = payloadZ.extend({
  key: keyZ.optional(),
  name: nameZ,
  leaseholder: zod.uint12.optional(),
  index: keyZ.optional(),
  isIndex: z.boolean().optional(),
  internal: z.boolean().default(false),
  virtual: z.boolean().default(false),
  expression: z.string().default(""),
  operations: array.nullishToEmpty(operationZ).optional(),
});

export interface New extends Omit<
  z.input<typeof newZ>,
  "dataType" | "status" | "internal"
> {
  dataType: CrudeDataType;
}

export const paramsZ = z.union([
  zod.toArray(keyZ),
  zod.toArray(nameZ),
  zod.toArray(payloadZ).transform((p) => p.map((c) => c.key)),
]);
export type Params = Key | Name | Keys | Names | Payload | Payload[];

export const ontologyID = ontology.createIDFactory<Key>("channel");
export const TYPE_ONTOLOGY_ID = ontologyID(0);

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
