// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod/v4";

import { ontology } from "@/ontology";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const stateZ = z.enum(["generating", "running", "error", "stopped"]);
export type State = z.infer<typeof stateZ>;

export const roleZ = z.enum(["user", "agent"]);
export type Role = z.infer<typeof roleZ>;

export const messageZ = z.object({
  role: roleZ,
  content: z.string(),
  time: z.union([z.number(), z.string().transform(Number)]).default(0),
});
export interface Message extends z.infer<typeof messageZ> {}

export const agentZ = z.object({
  key: keyZ,
  name: z.string(),
  messages: messageZ
    .array()
    .nullable()
    .transform((v) => v ?? []),
  arcKey: z.uuid().or(z.string()),
  rackKey: z.number().optional(),
  taskKey: z.number().optional(),
  state: stateZ.default("stopped"),
});
export interface Agent extends z.infer<typeof agentZ> {}

export const newZ = z.object({
  name: z.string().optional(),
  messages: messageZ.array().optional(),
});
export interface New extends z.input<typeof newZ> {}

export const ONTOLOGY_TYPE = "agent";
export type OntologyType = typeof ONTOLOGY_TYPE;

export const ontologyID = ontology.createIDFactory<Key>("agent");
export const TYPE_ONTOLOGY_ID = ontologyID("");
