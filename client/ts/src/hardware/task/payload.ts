// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

export const taskKeyZ = z.bigint().or(z.number()).transform((k) => k.toString())

export const taskZ = z.object({
  key: taskKeyZ,
  name: z.string(),
  type: z.string(),
  config: z.record(z.unknown()).or(z.string().transform((c) => JSON.parse(c))) as z.ZodType<UnknownRecord>,
});

export const newTaskZ = taskZ
  .omit({ key: true })
  .extend({ key: taskKeyZ.transform((k) => k.toString()).optional(), config: z.unknown().transform((c) => JSON.stringify(c)) });

export type NewTask = z.input<typeof newTaskZ>;

export type Task<
  T extends string = string, 
  C extends UnknownRecord = UnknownRecord
> = Omit<z.infer<typeof taskZ>, "config" | "type"> & { type: T, config: C };

