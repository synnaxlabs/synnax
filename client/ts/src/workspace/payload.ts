// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnknownRecord, unknownRecordZ } from "@synnaxlabs/x/record";
import { z } from "zod";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

// --- VERY IMPORTANT ---
// Synnax's encoders (in the binary package inside x) automatically convert the case
// of keys in objects to snake_case and back to camelCase when encoding and decoding
// respectively. This is done to ensure that the keys are consistent across all
// languages and platforms. Sometimes workspaces have keys that are uuids, which have
// dashes, and those get messed up. So we just use regular JSON for workspaces.
const parse = (s: string): UnknownRecord => JSON.parse(s);

export const workspaceZ = z.object({
  key: keyZ,
  name: z.string(),
  layout: unknownRecordZ.or(z.string().transform(parse)),
});
export interface Workspace extends z.infer<typeof workspaceZ> {}

export const ONTOLOGY_TYPE = "workspace";

export const newZ = workspaceZ
  .partial({ key: true })
  .transform((p) => ({ ...p, layout: JSON.stringify(p.layout) }));
export interface New extends z.input<typeof newZ> {}

export const remoteZ = workspaceZ
  .omit({ layout: true })
  .extend({ layout: z.string().transform(parse) });
