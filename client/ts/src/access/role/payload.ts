// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { ontology } from "@/ontology";

export const keyZ = z.uuid();

export type Key = z.infer<typeof keyZ>;

export const roleZ = z.object({
  key: keyZ,
  name: z.string(),
  description: z.string().optional(),
  internal: z.boolean().optional(),
});

export type Role = z.infer<typeof roleZ>;

export const newZ = roleZ.partial({ key: true });

export type New = z.infer<typeof newZ>;

export const ontologyID = ontology.createIDFactory<Key>("role");
export const TYPE_ONTOLOGY_ID = ontologyID("");
