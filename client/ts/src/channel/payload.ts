// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate } from "@synnaxlabs/x/telem";
import { z } from "zod";

import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";

export const keyZ = z.number();
export type Key = number;
export type Keys = number[];
export type Name = string;
export type Names = string[];
export type KeyOrName = Key | Name;
export type KeysOrNames = Keys | Names;
export type Params = Key | Name | Keys | Names;

export const payload = z.object({
  name: z.string(),
  key: z.number(),
  rate: Rate.z,
  dataType: DataType.z,
  leaseholder: z.number(),
  index: z.number(),
  isIndex: z.boolean(),
  internal: z.boolean(),
  virtual: z.boolean(),
  alias: z.string().optional(),
  expression: z.string().default(""),
  requires: nullableArrayZ(keyZ),
});

export type Payload = z.infer<typeof payload>;

export const newPayload = payload.extend({
  key: z.number().optional(),
  leaseholder: z.number().optional(),
  index: z.number().optional(),
  rate: Rate.z.optional().default(0),
  isIndex: z.boolean().optional(),
  internal: z.boolean().optional().default(false),
  virtual: z.boolean().optional().default(false),
  expression: z.string().optional().default(""),
  requires: nullableArrayZ(keyZ).optional().default([]),
});

export type NewPayload = z.input<typeof newPayload>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "channel";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key: key.toString() });
