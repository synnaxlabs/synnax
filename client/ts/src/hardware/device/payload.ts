// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, type UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

import { rack } from "@/hardware/rack";

export const keyZ = z.string();
export type Key = z.infer<typeof keyZ>;

export const deviceZ = z.object({
  key: keyZ,
  rack: rack.keyZ,
  name: z.string(),
  make: z.string(),
  model: z.string(),
  location: z.string(),
  configured: z.boolean().optional(),
  properties: z
    .record(z.unknown())
    .or(
      z
        .string()
        .transform(
          (c) => (c === "" ? {} : binary.JSON_CODEC.decodeString(c)) as UnknownRecord,
        ),
    ) as z.ZodType<UnknownRecord>,
});
export interface Device<P extends UnknownRecord = UnknownRecord>
  extends Omit<z.output<typeof deviceZ>, "properties"> {
  properties: P;
}

export const newZ = deviceZ.extend({
  properties: z.unknown().transform((c) => binary.JSON_CODEC.encodeString(c)),
});
export interface New<P extends UnknownRecord = UnknownRecord>
  extends Omit<z.input<typeof newZ>, "properties"> {
  properties: P;
}

export const ONTOLOGY_TYPE = "device";
