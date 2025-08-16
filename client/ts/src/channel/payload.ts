// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeDataType, DataType, status, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { nullableArrayZ } from "@/util/zod";

export const keyZ = z.number();
export type Key = z.infer<typeof keyZ>;
export type Keys = Key[];
export const nameZ = z.string();
export type Name = z.infer<typeof nameZ>;
export type Names = Name[];
export type KeyOrName = Key | Name;
export type KeysOrNames = Keys | Names;
export type PrimitiveParams = Key | Name | Keys | Names;

export const statusZ = status.statusZ();
export type Status = z.infer<typeof statusZ>;
export const payloadZ = z.object({
  name: nameZ,
  key: keyZ,
  dataType: DataType.z,
  leaseholder: z.number(),
  index: keyZ,
  isIndex: z.boolean(),
  internal: z.boolean(),
  virtual: z.boolean(),
  alias: z.string().optional(),
  expression: z.string().default(""),
  requires: nullableArrayZ(keyZ),
  status: statusZ.optional(),
});
export interface Payload extends z.infer<typeof payloadZ> {}

export const newZ = payloadZ.extend({
  key: keyZ.optional(),
  leaseholder: z.number().optional(),
  index: keyZ.optional(),
  isIndex: z.boolean().optional(),
  internal: z.boolean().optional().default(false),
  virtual: z.boolean().optional().default(false),
  expression: z.string().optional().default(""),
  requires: nullableArrayZ(keyZ).optional().default([]),
});

export interface New extends Omit<z.input<typeof newZ>, "dataType"> {
  dataType: CrudeDataType;
}

export const paramsZ = z.union([
  zod.toArray(keyZ),
  zod.toArray(nameZ),
  zod.toArray(payloadZ).transform((p) => p.map((c) => c.key)),
]);
export type Params = Key | Name | Keys | Names | Payload | Payload[];
