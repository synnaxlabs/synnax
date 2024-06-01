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

export type Params = Key | Key[];

export const labelZ = z.object({
  key: keyZ,
  name: z.string().min(1),
  color: z.string(),
});

export type Label = z.infer<typeof labelZ>;

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: "label", key });
