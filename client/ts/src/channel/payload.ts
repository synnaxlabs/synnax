// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional } from "@synnaxlabs/x";
import { type CrudeDataType, DataType } from "@synnaxlabs/x/telem";
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
export type Params = Key | Name | Keys | Names | Payload | Payload[];

export const channelZ = z.object({
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
});
export interface Payload extends z.infer<typeof channelZ> {}

export const newZ = channelZ.extend({
  key: keyZ.optional(),
  leaseholder: z.number().optional(),
  index: keyZ.optional(),
  isIndex: z.boolean().optional(),
  internal: z.boolean().optional().default(false),
  virtual: z.boolean().optional().default(false),
  expression: z.string().optional().default(""),
  requires: nullableArrayZ(keyZ).optional().default([]),
});

export interface New
  extends Omit<
    Optional<
      Payload,
      | "key"
      | "leaseholder"
      | "index"
      | "isIndex"
      | "internal"
      | "virtual"
      | "expression"
      | "requires"
    >,
    "dataType"
  > {
  dataType: CrudeDataType;
}

export const ONTOLOGY_TYPE = "channel";
export type OntologyType = typeof ONTOLOGY_TYPE;
