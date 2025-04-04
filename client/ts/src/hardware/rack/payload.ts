// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { zod } from "@synnaxlabs/x";
import { TimeStamp } from "@synnaxlabs/x/telem";
import { z } from "zod";

export const keyZ = zod.uint32;
export type Key = z.infer<typeof keyZ>;

export const rackStateZ = z.object({
  key: keyZ,
  heartbeat: z.number(),
  lastReceived: TimeStamp.z.or(z.number().transform((n) => new TimeStamp(n))),
});

export interface RackState {
  key: Key;
  heartbeat: number;
  lastReceived: TimeStamp;
}

export const rackZ = z.object({
  key: keyZ,
  name: z.string(),
  state: rackStateZ.optional(),
});

export interface Payload extends Omit<z.output<typeof rackZ>, "state"> {
  state?: RackState;
}

export const newZ = rackZ.partial({ key: true });
export interface New extends z.input<typeof newZ> {}

export const ONTOLOGY_TYPE = "rack";
export type OntologyType = typeof ONTOLOGY_TYPE;

export type Heartbeat = number;
