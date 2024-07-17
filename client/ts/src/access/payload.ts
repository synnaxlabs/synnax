// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const OntologyID = z.object({
  key: z.string().optional(),
  type: z.string().optional(),
});

export type OntologyIDType = z.infer<typeof OntologyID>

export const keyZ = z.string().uuid();

export type Key = z.infer<typeof keyZ>

export type Params = Key | Key[]

export const policyZ = z.object({
  key: keyZ,
  subjects: OntologyID.array(),
  objects: OntologyID.array(),
  actions: z.string().array(),
});

export type Policy = z.infer<typeof policyZ>
