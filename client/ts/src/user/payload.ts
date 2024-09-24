// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { ontology } from "@/ontology";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;

export const userZ = z.object({
  key: keyZ,
  username: z.string().min(1),
  firstName: z.string(),
  lastName: z.string(),
  rootUser: z.boolean(),
});
export type User = z.infer<typeof userZ>;

export const newUserZ = userZ
  .partial({ key: true, firstName: true, lastName: true })
  .omit({ rootUser: true })
  .extend({ password: z.string().min(1) });
export type NewUser = z.infer<typeof newUserZ>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "user";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
