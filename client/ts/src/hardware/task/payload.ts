// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, type observe, type UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

import type * as rack from "@/hardware/rack/payload";

export const keyZ = z.union([
  z.string(),
  z.bigint().transform((k) => k.toString()),
  z.number().transform((k) => k.toString()),
]);
export type Key = z.infer<typeof keyZ>;

export const stateZ = z.object({
  task: keyZ,
  variant: z.string(),
  key: z.string().optional(),
  details: z
    .record(z.unknown())
    .or(z.string().transform((c) => (c === "" ? {} : JSON.parse(c))))
    .or(z.array(z.unknown()))
    .or(z.null()),
});
export interface State<Details extends {} = UnknownRecord>
  extends Omit<z.infer<typeof stateZ>, "details"> {
  details?: Details;
}

export const taskZ = z.object({
  key: keyZ,
  name: z.string(),
  type: z.string(),
  internal: z.boolean().optional(),
  config: z
    .record(z.unknown())
    .or(
      z.string().transform((c) => (c === "" ? {} : binary.JSON_CODEC.decodeString(c))),
    ) as z.ZodType<UnknownRecord>,
  state: stateZ.optional().nullable(),
  snapshot: z.boolean().optional(),
});
export interface Payload<
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
> extends Omit<z.output<typeof taskZ>, "config" | "type" | "state"> {
  type: Type;
  config: Config;
  state?: State<Details> | null;
}

export const newZ = taskZ.omit({ key: true }).extend({
  key: keyZ.transform((k) => k.toString()).optional(),
  config: z.unknown().transform((c) => binary.JSON_CODEC.encodeString(c)),
});
export interface New<
  Config extends UnknownRecord = UnknownRecord,
  Type extends string = string,
> extends Omit<z.input<typeof newZ>, "config" | "state"> {
  type: Type;
  config: Config;
}

export const commandZ = z.object({
  task: keyZ,
  type: z.string(),
  key: z.string(),
  args: z
    .record(z.unknown())
    .or(z.string().transform((c) => (c === "" ? {} : JSON.parse(c))))
    .or(z.array(z.unknown()))
    .or(z.null())
    .optional() as z.ZodOptional<z.ZodType<UnknownRecord>>,
});
export interface Command<Args extends {} = UnknownRecord>
  extends Omit<z.infer<typeof commandZ>, "args"> {
  args?: Args;
}

export interface StateObservable<Details extends {} = UnknownRecord>
  extends observe.ObservableAsyncCloseable<State<Details>> {}

export interface CommandObservable extends observe.ObservableAsyncCloseable<Command> {}

export const ONTOLOGY_TYPE = "task";
export type OntologyType = typeof ONTOLOGY_TYPE;

export const getRackKey = (key: Key): rack.Key => Number(BigInt(key) >> BigInt(32));
