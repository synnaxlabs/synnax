// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { idZ } from "@/ontology/payload";

export const keyZ = z.string().uuid();

export type Key = z.infer<typeof keyZ>

export type Params = Key | Key[]

export const policyZ = z.object({
  key: keyZ,
  subjects: idZ.array(),
  objects: idZ.array(),
  actions: z.string().array(),
});
export type Policy = z.infer<typeof policyZ>

export const newPolicyPayloadZ = policyZ.extend({key: keyZ.optional()})
export type NewPolicyPayload = z.infer<typeof newPolicyPayloadZ>

