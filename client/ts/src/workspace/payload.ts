// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnknownRecord, unknownRecordZ } from "@synnaxlabs/x/record";
import { z } from "zod";

import { ontology } from "@/ontology";

export const keyZ = z.string().uuid();

export type Key = z.infer<typeof keyZ>;

export type Params = Key | Key[];

// --- VERY IMPORTANT ---
// Synnax's encoders (in the binary package inside x) automatically convert the case
// of keys in objects to snake_case and back to camelCase when encoding and decoding
// respectively. This is done to ensure that the keys are consistent across all
// languages and platforms. Sometimes workspaces have keys that are uuids, which have
// dashes, and those get messed up. So we just use regular JSON for workspaces.
const parse = (s: string): UnknownRecord => JSON.parse(s) as UnknownRecord;

export const workspaceZ = z.object({
  name: z.string(),
  key: keyZ,
  layout: unknownRecordZ.or(z.string().transform((s) => parse(s) as UnknownRecord)),
});

export const workspaceRemoteZ = workspaceZ.omit({ layout: true }).extend({
  layout: z.string().transform((s) => parse(s) as UnknownRecord),
});

export type Workspace = z.infer<typeof workspaceZ>;

export const WorkspaceOntologyType = "workspace" as ontology.ResourceType;

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: WorkspaceOntologyType, key: key });
