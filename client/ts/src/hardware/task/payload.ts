// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, observe, UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";

export const taskKeyZ = z.union([
  z.string(),
  z.bigint().transform((k) => k.toString()),
  z.number().transform((k) => k.toString()),
]);

export type TaskKey = z.infer<typeof taskKeyZ>;

export const stateZ = z.object({
  task: taskKeyZ,
  variant: z.string(),
  key: z.string().optional(),
  details: z
    .record(z.unknown())
    .or(
      z.string().transform((c) => {
        if (c === "") return {};
        return JSON.parse(c);
      }),
    )
    .or(z.array(z.unknown()))
    .or(z.null()),
});

export type State<D extends {} = UnknownRecord> = Omit<
  z.infer<typeof stateZ>,
  "details"
> & {
  details?: D;
};

export const taskZ = z.object({
  key: taskKeyZ,
  name: z.string(),
  type: z.string(),
  internal: z.boolean().optional(),
  config: z.record(z.unknown()).or(
    z.string().transform((c) => {
      if (c === "") return {};
      return binary.JSON_CODEC.decodeString(c);
    }),
  ) as z.ZodType<UnknownRecord>,
  state: stateZ.optional().nullable(),
  snapshot: z.boolean().optional(),
});

export const newTaskZ = taskZ.omit({ key: true }).extend({
  key: taskKeyZ.transform((k) => k.toString()).optional(),
  config: z.unknown().transform((c) => binary.JSON_CODEC.encodeString(c)),
});

export type NewTask<
  C extends UnknownRecord = UnknownRecord,
  T extends string = string,
> = Omit<z.input<typeof newTaskZ>, "config" | "state"> & {
  type: T;
  config: C;
};

export type Payload<
  C extends UnknownRecord = UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
> = Omit<z.output<typeof taskZ>, "config" | "type" | "state"> & {
  type: T;
  config: C;
  state?: State<D> | null;
};

export const commandZ = z.object({
  task: taskKeyZ,
  type: z.string(),
  key: z.string(),
  args: z.record(z.unknown()).or(
    z.string().transform((c) => {
      if (c === "") return {};
      return JSON.parse(c);
    }),
  ) as z.ZodType<UnknownRecord>,
});

export type StateObservable<D extends UnknownRecord = UnknownRecord> =
  observe.ObservableAsyncCloseable<State<D>>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "task";

export const ontologyID = (key: TaskKey): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key: key.toString() });
