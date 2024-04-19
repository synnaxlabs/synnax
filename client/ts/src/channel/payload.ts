// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate } from "@synnaxlabs/x";
import { z } from "zod";

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
  alias: z.string().optional(),
});

export type Payload = z.infer<typeof payload>;

export const newPayload = payload.extend({
  key: z.number().optional(),
  leaseholder: z.number().optional(),
  index: z.number().optional(),
  rate: Rate.z.optional().default(0),
  isIndex: z.boolean().optional(),
});

export type NewPayload = z.input<typeof newPayload>;
